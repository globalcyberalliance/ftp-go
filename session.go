// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"net"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

const (
	defaultWelcomeMessage = "Welcome to the Go FTP Server"
)

type (
	// Context represents a context the driver may want to know
	Context struct {
		Sess  *Session
		Data  map[string]interface{} // share data between middlewares
		Cmd   string                 // request command on this request
		Param string                 // request param on this request
	}

	// Session represents a session between ftp client and the server
	Session struct {
		dataConn      DataSocket
		Conn          net.Conn
		Ctx           context.Context
		controlReader *bufio.Reader
		controlWriter *bufio.Writer
		server        *Server
		Data          map[string]interface{} // shared data between different commands
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

// RemoteAddr returns the remote ftp client's address
func (sess *Session) RemoteAddr() net.Addr {
	return sess.Conn.RemoteAddr()
}

// LoginUser returns the login user name if login
func (sess *Session) LoginUser() string {
	return sess.user
}

// IsLogin returns if user has login
func (sess *Session) IsLogin() bool {
	return len(sess.user) > 0
}

// PublicIP returns the public ip of the server
func (sess *Session) PublicIP() string {
	return sess.server.PublicIP
}

// Options returns the server options
func (sess *Session) Options() *Options {
	return sess.server.Options
}

// Server returns the server of session
func (sess *Session) Server() *Server {
	return sess.server
}

// DataConn returns the data connection
func (sess *Session) DataConn() DataSocket {
	return sess.dataConn
}

func (sess *Session) passiveListenIP() string {
	var listenIP string
	if len(sess.PublicIP()) > 0 {
		listenIP = sess.PublicIP()
	} else {
		listenIP = sess.Conn.LocalAddr().(*net.TCPAddr).IP.String()
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
	if len(sess.server.PassivePorts) > 0 {
		portRange := strings.Split(sess.server.PassivePorts, "-")

		if len(portRange) != 2 {
			log.Println("empty port")
			return 0
		}

		minPort, _ := strconv.Atoi(strings.TrimSpace(portRange[0]))
		maxPort, _ := strconv.Atoi(strings.TrimSpace(portRange[1]))

		return minPort + mrand.Intn(maxPort-minPort)
	}

	// Let system automatically choose one port.
	return 0
}

// newSessionID returns a random 20 char string that can be used as a unique session ID.
func newSessionID() string {
	hash := sha256.New()
	_, err := io.CopyN(hash, rand.Reader, 50)
	if err != nil {
		return "????????????????????"
	}
	md := hash.Sum(nil)
	mdStr := hex.EncodeToString(md)
	return mdStr[0:20]
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
	sess.writeMessage(220, sess.server.WelcomeMessage)

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
		}, line)

		sess.receiveLine(line)

		// QUIT command closes connection, break to avoid error on reading from a closed socket.
		if sess.closed {
			break
		}
	}

	sess.log("Connection Terminated")
}

// Close will manually close this connection, even if the client isn't ready.
func (sess *Session) Close() {
	sess.Conn.Close()
	sess.closed = true
	sess.reqUser = ""
	sess.user = ""
	if sess.dataConn != nil {
		sess.dataConn.Close()
		sess.dataConn = nil
	}
}

func (sess *Session) upgradeToTLS() error {
	sess.log("Upgrading connection to TLS")

	tlsConn := tls.Server(sess.Conn, sess.server.tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return err
	}

	sess.Conn = tlsConn
	sess.controlReader = bufio.NewReader(tlsConn)
	sess.controlWriter = bufio.NewWriter(tlsConn)
	sess.tls = true

	return nil
}

// receiveLine accepts a single line FTP command and co-ordinates an
// appropriate response.
func (sess *Session) receiveLine(line string) {
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 1<<16)
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
	if !ok {
		sess.writeMessage(500, "Command not found")
		return
	}

	if cmdObj.RequireParam() && param == "" {
		sess.writeMessage(553, "action aborted, required param missing")
	} else if sess.server.Options.ForceTLS && !sess.tls && !(cmdObj == sess.server.Commands["AUTH"] && param == "TLS") {
		sess.writeMessage(534, "Request denied for policy reasons. AUTH TLS required.")
	} else if cmdObj.RequireAuth() && sess.user == "" {
		sess.writeMessage(530, "not logged in")
	} else {
		cmdObj.Execute(sess, param)
		sess.preCommand = cmdGiven
	}
}

func (sess *Session) parseLine(line string) (string, string) {
	params := strings.SplitN(strings.Trim(line, "\r\n"), " ", 2)
	if len(params) == 0 {
		return "", ""
	}

	if len(params) == 1 {
		return params[0], ""
	}

	return params[0], params[1]
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
func (sess *Session) buildPath(filename string) (fullPath string) {
	if len(filename) > 0 && filename[0:1] == "/" {
		fullPath = filepath.Clean(filename)
	} else if len(filename) > 0 && filename != "-a" {
		fullPath = filepath.Clean(sess.curDir + "/" + filename)
	} else {
		fullPath = filepath.Clean(sess.curDir)
	}
	fullPath = strings.Replace(fullPath, "//", "/", -1)
	fullPath = strings.Replace(fullPath, string(filepath.Separator), "/", -1)
	return
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
	sess.writeMessage(226, message)
}

func (sess *Session) sendOutofBandDataWriter(data io.ReadCloser) error {
	bytes, err := io.Copy(sess.dataConn, data)
	if err != nil {
		sess.dataConn.Close()
		sess.dataConn = nil
		return err
	}

	message := "Closing data connection, sent " + strconv.Itoa(int(bytes)) + " bytes"
	sess.writeMessage(226, message)
	sess.dataConn.Close()
	sess.dataConn = nil

	return nil
}

func (sess *Session) changeCurDir(path string) error {
	sess.curDir = path
	return nil
}

func (sess *Session) log(message interface{}) {
	sess.server.logger.Print(sess.id, message)
}

func (sess *Session) logf(format string, v ...interface{}) {
	sess.server.logger.Printf(sess.id, format, v...)
}
