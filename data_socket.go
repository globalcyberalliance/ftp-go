// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/globalcyberalliance/ftp-go/ratelimit"
)

type (
	// DataSocket describes a data socket is used to send non-control data between the client and server.
	DataSocket interface {
		Host() string

		Port() int

		// Read implements the standard io.Reader interface.
		Read(p []byte) (n int, err error)

		// ReadFrom implements the standard io.ReaderFrom interface.
		ReadFrom(r io.Reader) (int64, error)

		// Write implements the standard io.Writer interface.
		Write(p []byte) (n int, err error)

		// Close implements the standard io.Closer interface.
		Close() error
	}

	activeSocket struct {
		conn   *net.TCPConn
		host   string
		port   int
		reader io.Reader
		writer io.Writer
		sess   *Session
	}
)

func newActiveSocket(sess *Session, remote string, port int) (*activeSocket, error) {
	connectTo := net.JoinHostPort(remote, strconv.Itoa(port))

	sess.log("Opening active data connection to " + connectTo)

	raddr, err := net.ResolveTCPAddr("tcp", connectTo)
	if err != nil {
		sess.log(err)
		return nil, fmt.Errorf("resolve tcp: %w", err)
	}

	tcpConn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		sess.log(err)
		return nil, fmt.Errorf("dial tcp: %w", err)
	}

	socket := &activeSocket{
		conn:   tcpConn,
		host:   remote,
		port:   port,
		reader: ratelimit.Reader(tcpConn, sess.server.rateLimiter),
		writer: ratelimit.Writer(tcpConn, sess.server.rateLimiter),
		sess:   sess,
	}

	return socket, nil
}

func (socket *activeSocket) Host() string {
	return socket.host
}

func (socket *activeSocket) Port() int {
	return socket.port
}

func (socket *activeSocket) Read(p []byte) (int, error) {
	bRead, err := socket.reader.Read(p)
	if err != nil {
		return 0, fmt.Errorf("read: %w", err)
	}

	return bRead, nil
}

func (socket *activeSocket) ReadFrom(r io.Reader) (int64, error) {
	bRead, err := io.Copy(socket.writer, r)
	if err != nil {
		return 0, fmt.Errorf("read: %w", err)
	}

	return bRead, nil
}

func (socket *activeSocket) Write(p []byte) (int, error) {
	bWritten, err := socket.writer.Write(p)
	if err != nil {
		return 0, fmt.Errorf("write: %w", err)
	}

	return bWritten, nil
}

func (socket *activeSocket) Close() error {
	if err := socket.conn.Close(); err != nil {
		return fmt.Errorf("close active socket: %w", err)
	}

	return nil
}

type passiveSocket struct {
	conn    net.Conn
	reader  io.Reader
	writer  io.Writer
	err     error
	sess    *Session
	ingress chan []byte
	egress  chan []byte
	host    string
	port    int
	lock    sync.Mutex // protects conn and err
}

// isErrorAddressAlreadyInUse detects if an error is "bind: address already in use"
//
// Originally from https://stackoverflow.com/a/52152912/164234
func isErrorAddressAlreadyInUse(err error) bool {
	errOpError := &net.OpError{}
	if !errors.As(err, &errOpError) {
		return false
	}

	var errSyscallError *os.SyscallError
	if !errors.As(errOpError.Err, &errSyscallError) {
		return false
	}

	var errErrno syscall.Errno
	if !errors.As(errSyscallError.Err, &errErrno) {
		return false
	}

	if errors.Is(errErrno, syscall.EADDRINUSE) {
		return true
	}

	const errAddrInUse = 10048
	if runtime.GOOS == "windows" && errErrno == errAddrInUse {
		return true
	}

	return false
}

func (sess *Session) newPassiveSocket() (*passiveSocket, error) {
	socket := &passiveSocket{
		sess:    sess,
		ingress: make(chan []byte),
		egress:  make(chan []byte),
		host:    sess.passiveListenIP(),
	}
	sess.dataConn = socket

	const retries = 10
	var err error
	for range retries {
		socket.port = sess.PassivePort()

		if err = socket.ListenAndServe(); err != nil && socket.port != 0 && isErrorAddressAlreadyInUse(err) {
			// Choose a different port on error already in use.
			continue
		}

		break
	}

	return socket, err
}

func (socket *passiveSocket) Host() string {
	return socket.host
}

func (socket *passiveSocket) Port() int {
	return socket.port
}

func (socket *passiveSocket) Read(p []byte) (int, error) {
	socket.lock.Lock()
	defer socket.lock.Unlock()

	if socket.err != nil {
		return 0, socket.err
	}

	return socket.reader.Read(p)
}

func (socket *passiveSocket) ReadFrom(r io.Reader) (int64, error) {
	socket.lock.Lock()
	defer socket.lock.Unlock()

	if socket.err != nil {
		return 0, socket.err
	}

	// io.Copy optimizes for TCPConn by using the sendfile syscall, falling back to standard read/write otherwise.
	return io.Copy(socket.writer, r)
}

func (socket *passiveSocket) Write(p []byte) (int, error) {
	socket.lock.Lock()
	defer socket.lock.Unlock()

	if socket.err != nil {
		return 0, socket.err
	}

	bWritten, err := socket.writer.Write(p)
	if err != nil {
		return 0, fmt.Errorf("write: %w", err)
	}

	return bWritten, nil
}

func (socket *passiveSocket) Close() error {
	socket.lock.Lock()
	defer socket.lock.Unlock()

	if socket.conn != nil {
		if err := socket.conn.Close(); err != nil {
			return fmt.Errorf("close socket: %w", err)
		}
	}

	return nil
}

func (socket *passiveSocket) ListenAndServe() error {
	laddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("", strconv.Itoa(socket.port)))
	if err != nil {
		socket.sess.log(err)
		return fmt.Errorf("resolve tcp address: %w", err)
	}

	var tcplistener *net.TCPListener
	tcplistener, err = net.ListenTCP("tcp", laddr)
	if err != nil {
		socket.sess.log(err)
		return fmt.Errorf("listen tcp: %w", err)
	}

	// The timeout, for a remote client to establish connection with a PASV style data connection.
	const acceptTimeout = 60 * time.Second
	err = tcplistener.SetDeadline(time.Now().Add(acceptTimeout))
	if err != nil {
		socket.sess.log(err)
		return fmt.Errorf("set deadline: %w", err)
	}

	var listener net.Listener = tcplistener
	add := listener.Addr()
	parts := strings.Split(add.String(), ":")
	port, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		socket.sess.log(err)
		return fmt.Errorf("parse port: %w", err)
	}

	socket.port = port
	if socket.sess.server.TLSConfig != nil {
		listener = tls.NewListener(listener, socket.sess.server.TLSConfig)
	}

	socket.lock.Lock()

	go func() {
		defer socket.lock.Unlock()

		conn, lErr := listener.Accept()
		if err != nil {
			socket.err = lErr
			return
		}

		socket.err = nil
		socket.conn = conn
		socket.reader = ratelimit.Reader(socket.conn, socket.sess.server.rateLimiter)
		socket.writer = ratelimit.Writer(socket.conn, socket.sess.server.rateLimiter)

		_ = listener.Close()
	}()

	return nil
}
