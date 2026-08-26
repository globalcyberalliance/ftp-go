// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout        = 60 * time.Second
	defaultWelcomeMessage = "Welcome to the Go FTP Server"
)

type (
	// Context represents a context the driver may want to know.
	Context struct {
		CMD   string         // Request command on this request.
		Data  map[string]any // Share data between middleware.
		Param string         // Request param on this request.
		Sess  *Session
	}

	// Session represents a session between an FTP client and the server.
	Session struct {
		ctx           context.Context
		dataConn      DataSocket
		Conn          net.Conn
		controlReader *bufio.Reader
		controlWriter *bufio.Writer
		server        *Server
		Data          map[string]any // Shared data between different commands.
		id            string
		curDir        string
		reqUser       string
		user          string
		renameFrom    string
		preCommand    string
		clientSoft    string
		lastFilePos   int64
		closed        bool
		tls           bool
	}
)

func newContext(sess *Session, cmd string, param string, data map[string]any) *Context {
	if data == nil {
		data = make(map[string]any)
	}

	return &Context{
		CMD:   cmd,
		Data:  data,
		Param: param,
		Sess:  sess,
	}
}

func (sess *Session) Context() context.Context {
	if sess.ctx != nil {
		return sess.ctx
	}

	return context.Background()

}

// LocalAddr returns the local FTP server's address.
func (sess *Session) LocalAddr() net.Addr {
	return sess.Conn.LocalAddr()
}

// RemoteAddr returns the remote FTP client's address.
func (sess *Session) RemoteAddr() net.Addr {
	return sess.Conn.RemoteAddr()
}

// LoginUser returns the logged in user's name.
func (sess *Session) LoginUser() string {
	return sess.user
}

// IsLogin returns if the user has logged in.
func (sess *Session) IsLogin() bool {
	return len(sess.user) > 0
}

// PublicIP returns the public ip of the server.
func (sess *Session) PublicIP() string {
	return sess.server.PublicIP
}

// Options returns the server options.
func (sess *Session) Options() *Options {
	return sess.server.Options
}

// Server returns the server of the session.
func (sess *Session) Server() *Server {
	return sess.server
}

// DataConn returns the data connection.
func (sess *Session) DataConn() DataSocket {
	return sess.dataConn
}

func (sess *Session) passiveListenIP() string {
	var listenIP string

	tcpAddr, ok := sess.Conn.LocalAddr().(*net.TCPAddr)
	if ok {
		listenIP = tcpAddr.IP.String()
	}
	if len(sess.PublicIP()) > 0 {
		listenIP = sess.PublicIP()
	}

	if listenIP == "::1" {
		return listenIP
	}

	lastIdx := strings.LastIndex(listenIP, ":")
	if lastIdx <= 0 {
		return listenIP
	}

	return listenIP[:lastIdx]
}

// PassivePort returns the port which could be used by passive mode.
func (sess *Session) PassivePort() int {
	if len(sess.server.PassivePorts) == 0 {
		// Let the system automatically choose one port.
		return 0
	}

	// Port range format: "min-max".
	const expectedRangeParts = 2

	portRange := strings.Split(sess.server.PassivePorts, "-")
	if len(portRange) != expectedRangeParts {
		log.Println("invalid port range format: expected min-max")
		return 0
	}

	// TODO: Handle errors.
	minPort, _ := strconv.Atoi(strings.TrimSpace(portRange[0]))
	maxPort, _ := strconv.Atoi(strings.TrimSpace(portRange[1]))

	rangeSize := maxPort - minPort
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(rangeSize)))

	return minPort + int(n.Int64())
}

// newSessionID returns a random 20-character hexadecimal string that can be used as a unique session ID.
func newSessionID() (string, error) {
	const hexCharsPerByte = 2 // Each byte encodes to 2 hex characters.
	const sessionIDLength = 20

	bytesNeeded := sessionIDLength / hexCharsPerByte
	randomBytes := make([]byte, bytesNeeded)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}

	return hex.EncodeToString(randomBytes), nil
}

// Serve starts an endless loop that reads FTP commands from the client and
// responds appropriately. terminated is a channel that will receive a true
// message when the connection closes. This loop will be running inside a
// goroutine, so use this channel to be notified when the connection can be
// cleaned up.
func (sess *Session) Serve() {
	defer sess.Close()

	// Leave a slight delay to close the context (needed to allow the connection to gracefully close).
	defer func() {
		if recovery := recover(); recovery != nil {
			sess.log(fmt.Sprintf("recovered from handle panic; recovered=%v; stack=%v", recovery, string(debug.Stack())))
		}
	}()

	sess.log("Connection Established")
	sess.writeMessage(CodeServiceReady, sess.server.WelcomeMessage)

	// Read commands.
	for {
		line, err := sess.controlReader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				sess.log(fmt.Sprint("Read error:", err))
			}

			break
		}

		sess.server.notifiers.BeforeCommand(&Context{
			Sess: sess,
		}, strings.Trim(line, "\r\n"))

		sess.receiveLine(line)

		// QUIT command closes connection, break to avoid error on reading from a closed socket.
		if sess.closed {
			break
		}
	}

	sess.log("Connection Terminated")
}

// Close will manually close this connection, even if the client isn't ready.
func (sess *Session) Close() error {
	sess.closed = true
	sess.reqUser = ""
	sess.user = ""

	// Close the data connection first so a failure closing the control connection can't strand it.
	if dataConn := sess.dataConn; dataConn != nil {
		sess.dataConn = nil

		if err := dataConn.Close(); err != nil {
			return fmt.Errorf("close data connection: %w", err)
		}
	}

	if err := sess.Conn.Close(); err != nil {
		return fmt.Errorf("close connection: %w", err)
	}

	return nil
}

func (sess *Session) upgradeToTLS() error {
	sess.log("Upgrading connection to TLS")

	tlsConn := tls.Server(sess.Conn, sess.server.TLSConfig)
	if err := tlsConn.HandshakeContext(sess.Context()); err != nil {
		return fmt.Errorf("tls handshake: %w", err)
	}

	sess.Conn = tlsConn
	sess.controlReader = bufio.NewReader(tlsConn)
	sess.controlWriter = bufio.NewWriter(tlsConn)
	sess.tls = true

	return nil
}

// receiveLine accepts a single line FTP command and co-ordinates an appropriate response.
func (sess *Session) receiveLine(line string) {
	defer func() {
		if err := recover(); err != nil {
			// 64KB buffer for stack trace.
			const stackBufferSize = 64 * 1024

			buf := make([]byte, stackBufferSize)
			buf = buf[:runtime.Stack(buf, false)]
			sess.logf("handler crashed with error:%v\n%s", err, buf)
		}
	}()

	command, param := sess.parseLine(line)
	cmdGiven := strings.ToUpper(command)
	sess.server.Logger.PrintCommand(sess.id, command, param)

	sess.server.CommandsMu.RLock()
	defer sess.server.CommandsMu.RUnlock()

	cmdObj, ok := sess.server.Commands[cmdGiven]
	sess.server.notifiers.AfterCommand(&Context{Sess: sess}, strings.Trim(line, "\r\n"), ok)
	if !ok {
		sess.writeMessage(CodeSyntaxError, "Command not found")
		return
	}

	if cmdObj.RequireParam() && param == "" {
		sess.writeMessage(CodeFileNameNotAllowed, "action aborted, required param missing")
		return
	}

	if sess.server.Options.ForceTLS && !sess.tls && (cmdObj != sess.server.Commands["AUTH"] || param != "TLS") {
		sess.writeMessage(CodeRequestDeniedPolicy, "Request denied for policy reasons. AUTH TLS required.")
		return
	}

	if cmdObj.RequireAuth() && sess.user == "" {
		sess.writeMessage(CodeNotLoggedIn, "not logged in")
		return
	}

	cmdObj.Execute(sess, param)
	sess.preCommand = cmdGiven
}

// parseLine splits a command line into command and parameters.
// It separates the first word (command) from the rest of the line (parameters).
func (sess *Session) parseLine(line string) (string, string) {
	// Split into command and parameters.
	const maxParts = 2

	parts := strings.SplitN(strings.Trim(line, "\r\n"), " ", maxParts)
	if len(parts) == 0 {
		return "", ""
	}

	command := parts[0]
	parameters := ""

	if len(parts) == maxParts {
		parameters = parts[1]
	}

	return command, parameters
}

func (sess *Session) WriteMessage(code int, message string) {
	sess.writeMessage(code, message)
}

// writeMessage will send a standard FTP response back to the client.
func (sess *Session) writeMessage(code int, message string) {
	sess.server.Logger.PrintResponse(sess.id, code, message)
	line := fmt.Sprintf("%d %s\r\n", code, message)
	_, _ = sess.controlWriter.WriteString(line)
	sess.controlWriter.Flush()
}

// writeMessage will send a standard FTP response back to the client.
func (sess *Session) writeMessageMultiline(code int, message string) {
	sess.server.Logger.PrintResponse(sess.id, code, message)
	line := fmt.Sprintf("%d-%s\r\n%d END\r\n", code, message, code)
	_, _ = sess.controlWriter.WriteString(line)
	sess.controlWriter.Flush()
}

func (sess *Session) BuildPath(filename string) string {
	return sess.buildPath(filename)
}

// buildPath takes a client supplied path or filename and generates a safe
// absolute path within their account sandbox.
//
//	buildpath("/")
//	=> "/"
//	buildpath("one.txt")
//	=> "/one.txt"
//	buildpath("/files/two.txt")
//	=> "/files/two.txt"
//	buildpath("files/two.txt")
//	=> "/files/two.txt"
//	buildpath("/../../../../etc/passwd")
//	=> "/etc/passwd"
//
// The driver implementation is responsible for deciding how to treat this path. They must not read the path off disk.
// They probably want to prefix the path with something to scope the users access to a sandbox.
func (sess *Session) buildPath(filename string) string {
	fullPath := filepath.Clean(sess.curDir)
	if len(filename) > 0 && filename[0:1] == "/" {
		fullPath = filepath.Clean(filename)
	} else if len(filename) > 0 && filename != "-a" {
		fullPath = filepath.Clean(sess.curDir + "/" + filename)
	}

	fullPath = strings.ReplaceAll(fullPath, "//", "/")
	fullPath = strings.ReplaceAll(fullPath, string(filepath.Separator), "/")

	return fullPath
}

// sendOutofbandData will send a string to the client via the currently open
// data socket. Assumes the socket is open and ready to be used.
func (sess *Session) sendOutofbandData(data []byte) {
	bytes := len(data)
	if sess.dataConn != nil {
		_, _ = sess.dataConn.Write(data)
		sess.dataConn.Close()
		sess.dataConn = nil
	}
	message := "Closing data connection, sent " + strconv.Itoa(bytes) + " bytes"
	sess.writeMessage(CodeClosingDataConnection, message)
}

// sendOutOfBandDataWriter sends data through the out-of-band data connection.
func (sess *Session) sendOutOfBandDataWriter(data io.ReadCloser) (err error) {
	defer func() {
		if sess.dataConn != nil {
			if closeErr := sess.dataConn.Close(); closeErr != nil && err == nil {
				err = fmt.Errorf("close data connection: %w", closeErr)
			}
			sess.dataConn = nil
		}
	}()

	bytesWritten, writeErr := io.Copy(sess.dataConn, data)
	if writeErr != nil {
		return fmt.Errorf("write out-of-band data: %w", writeErr)
	}

	message := fmt.Sprintf("Closing data connection, sent %d bytes", bytesWritten)
	sess.writeMessage(CodeClosingDataConnection, message)

	return nil
}

// changeCurDir changes the current directory after validating it exists.
func (sess *Session) changeCurDir(path string) error {
	fInfo, err := sess.server.Driver.Stat(&Context{Sess: sess}, path)
	if err != nil {
		return fmt.Errorf("stat directory: %w", err)
	}

	if !fInfo.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	sess.curDir = path

	return nil
}

func (sess *Session) log(message any) {
	sess.server.logger.Print(sess.id, message)
}

func (sess *Session) logf(format string, v ...any) {
	sess.server.logger.Printf(sess.id, format, v...)
}
