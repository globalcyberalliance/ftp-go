// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// Command represents a Command interface to a ftp command.
type Command interface {
	IsExtend() bool
	RequireParam() bool
	RequireAuth() bool
	Execute(*Session, string)
}

// DefaultCommands returns the default commands.
func DefaultCommands() map[string]Command {
	return map[string]Command{
		"ADAT": commandADAT{},
		"ALLO": commandALLO{},
		"APPE": commandAPPE{},
		"AUTH": commandAUTH{},
		"CDUP": commandCDUP{},
		"CWD":  commandCWD{},
		"CCC":  commandCCC{},
		"CONF": commandCONF{},
		"CLNT": commandCLNT{},
		"DELE": commandDELE{},
		"ENC":  commandENC{},
		"EPRT": commandEPRT{},
		"EPSV": commandEPSV{},
		"FEAT": commandFEAT{},
		"LIST": commandLIST{},
		"LPRT": commandLPRT{},
		"NLST": commandNLST{},
		"MDTM": commandMDTM{},
		"MIC":  commandMIC{},
		"MLSD": commandMLSD{},
		"MKD":  commandMKD{},
		"MODE": commandMODE{},
		"NOOP": commandNOOP{},
		"OPTS": commandOPTS{},
		"PASS": commandPASS{},
		"PASV": commandPASV{},
		"PBSZ": commandPBSZ{},
		"PORT": commandPORT{},
		"PROT": commandPROT{},
		"PWD":  commandPWD{},
		"QUIT": commandQUIT{},
		"RETR": commandRETR{},
		"REST": commandREST{},
		"RNFR": commandRNFR{},
		"RNTO": commandRNTO{},
		"RMD":  commandRMD{},
		"SIZE": commandSIZE{},
		"STAT": commandSTAT{},
		"STOR": commandSTOR{},
		"STRU": commandSTRU{},
		"SYST": commandSYST{},
		"TYPE": commandTYPE{},
		"USER": commandUSER{},
		"XCUP": commandCDUP{},
		"XCWD": commandCWD{},
		"XMKD": commandMKD{},
		"XPWD": commandPWD{},
		"XRMD": commandXRmd{},
	}
}

// commandALLO represents the ALLO FTP command, which is obsolete and not implemented in this server.
type commandALLO struct{}

func (cmd commandALLO) IsExtend() bool {
	return false
}

func (cmd commandALLO) RequireParam() bool {
	return false
}

func (cmd commandALLO) RequireAuth() bool {
	return false
}

func (cmd commandALLO) Execute(sess *Session, _ string) {
	sess.writeMessage(CodeCommandNotImplemented, "Obsolete")
}

// commandAPPE represents the APPE FTP command for appending data to an existing file or creating a new one.
type commandAPPE struct{}

func (cmd commandAPPE) IsExtend() bool {
	return false
}

func (cmd commandAPPE) RequireParam() bool {
	return true
}

func (cmd commandAPPE) RequireAuth() bool {
	return true
}

func (cmd commandAPPE) Execute(sess *Session, param string) {
	targetPath := sess.buildPath(param)
	sess.writeMessage(CodeFileStatusOK, "Data transfer starting")

	if sess.preCommand != "REST" {
		sess.lastFilePos = -1
	}
	defer func() {
		sess.lastFilePos = -1
	}()

	ctx := Context{
		Sess:  sess,
		CMD:   "APPE",
		Param: param,
		Data:  make(map[string]any),
	}
	sess.server.notifiers.BeforePutFile(&ctx, targetPath)
	size, err := sess.server.Driver.PutFile(&ctx, targetPath, sess.dataConn, sess.lastFilePos)
	sess.server.notifiers.AfterFilePut(&ctx, targetPath, size, err)
	if err != nil {
		sess.writeMessage(CodeFileActionNotTaken, fmt.Sprint("error during transfer: ", err))
		return
	}

	msg := fmt.Sprintf("OK, received %d bytes", size)
	sess.writeMessage(CodeClosingDataConnection, msg)
}

type commandCLNT struct{}

func (cmd commandCLNT) IsExtend() bool {
	return true
}

func (cmd commandCLNT) RequireParam() bool {
	return false
}

func (cmd commandCLNT) RequireAuth() bool {
	return false
}

func (cmd commandCLNT) Execute(sess *Session, param string) {
	sess.clientSoft = param
	sess.writeMessage(CodeCommandOK, "OK")
}

type commandOPTS struct{}

func (cmd commandOPTS) IsExtend() bool {
	return false
}

func (cmd commandOPTS) RequireParam() bool {
	return false
}

func (cmd commandOPTS) RequireAuth() bool {
	return false
}

func (cmd commandOPTS) Execute(sess *Session, param string) {
	const expectedParts = 2

	parts := strings.Fields(param)
	if len(parts) != expectedParts {
		sess.writeMessage(CodeFileUnavailable, "Unknow params")
		return
	}

	if strings.ToUpper(parts[0]) != "UTF8" {
		sess.writeMessage(CodeFileUnavailable, "Unknow params")
		return
	}

	if strings.ToUpper(parts[1]) == "ON" {
		sess.writeMessage(CodeCommandOK, "UTF8 mode enabled")
		return
	}

	sess.writeMessage(CodeFileUnavailable, "Unsupported non-utf8 mode")
}

type commandFEAT struct{}

func (cmd commandFEAT) IsExtend() bool {
	return false
}

func (cmd commandFEAT) RequireParam() bool {
	return false
}

func (cmd commandFEAT) RequireAuth() bool {
	return false
}

func (cmd commandFEAT) Execute(sess *Session, _ string) {
	sess.writeMessageMultiline(CodeSystemStatus, sess.server.feats)
}

// commandCDUP responds to the CDUP FTP command.
//
// Allows the client to change their current directory to the parent.
type commandCDUP struct{}

func (cmd commandCDUP) IsExtend() bool {
	return false
}

func (cmd commandCDUP) RequireParam() bool {
	return false
}

func (cmd commandCDUP) RequireAuth() bool {
	return true
}

func (cmd commandCDUP) Execute(sess *Session, _ string) {
	otherCmd := &commandCWD{}
	otherCmd.Execute(sess, "..")
}

// commandCWD responds to the CWD FTP command. It allows the client to change the
// current working directory.
type commandCWD struct{}

func (cmd commandCWD) IsExtend() bool {
	return false
}

func (cmd commandCWD) RequireParam() bool {
	return true
}

func (cmd commandCWD) RequireAuth() bool {
	return true
}

func (cmd commandCWD) Execute(sess *Session, param string) {
	buildPath := sess.buildPath(param)
	ctx := Context{
		Sess:  sess,
		CMD:   "CWD",
		Param: param,
		Data:  make(map[string]any),
	}
	info, err := sess.server.Driver.Stat(&ctx, buildPath)
	if err != nil {
		sess.logf("%v", err)
		sess.writeMessage(CodeFileUnavailable, fmt.Sprint("Directory change to ", buildPath, " failed."))
		return
	}
	if !info.IsDir() {
		sess.writeMessage(CodeFileUnavailable, fmt.Sprint("Directory change to ", buildPath, " is a file"))
		return
	}

	sess.server.notifiers.BeforeChangeCurDir(&ctx, sess.curDir, buildPath)
	err = sess.changeCurDir(buildPath)
	sess.server.notifiers.AfterCurDirChanged(&ctx, sess.curDir, buildPath, err)
	if err != nil {
		sess.logf("%v", err)
		sess.writeMessage(CodeFileUnavailable, fmt.Sprint("Directory change to ", buildPath, " failed."))
		return
	}

	sess.writeMessage(CodeFileActionOK, "Directory changed to "+buildPath)
}

// commandDELE responds to the DELE FTP command. It allows the client to delete a file.
type commandDELE struct{}

func (cmd commandDELE) IsExtend() bool {
	return false
}

func (cmd commandDELE) RequireParam() bool {
	return true
}

func (cmd commandDELE) RequireAuth() bool {
	return true
}

func (cmd commandDELE) Execute(sess *Session, param string) {
	buildPath := sess.buildPath(param)
	ctx := Context{
		Sess:  sess,
		CMD:   "DELE",
		Param: param,
		Data:  make(map[string]any),
	}
	sess.server.notifiers.BeforeDeleteFile(&ctx, buildPath)
	err := sess.server.Driver.DeleteFile(&ctx, buildPath)
	sess.server.notifiers.AfterFileDeleted(&ctx, buildPath, err)
	if err == nil {
		sess.writeMessage(CodeFileActionOK, "File deleted")
	} else {
		sess.logf("%v", err)
		sess.writeMessage(CodeFileUnavailable, "File delete failed. ")
	}
}

// commandEPRT responds to the EPRT FTP command. It allows the client to
// request an active data socket with more options than the original PORT
// command. It mainly adds ipv6 support.
type commandEPRT struct{}

func (cmd commandEPRT) IsExtend() bool {
	return true
}

func (cmd commandEPRT) RequireParam() bool {
	return true
}

func (cmd commandEPRT) RequireAuth() bool {
	return true
}

func (cmd commandEPRT) Execute(sess *Session, param string) {
	delim := param[0:1]
	parts := strings.Split(param, delim)

	addressFamily, err := strconv.Atoi(parts[1])
	if err != nil {
		sess.writeMessage(CodeNetworkProtocolNotSupported, "Network protocol not supported, use (1,2)")
		return
	}
	if addressFamily != 1 && addressFamily != 2 {
		sess.writeMessage(CodeNetworkProtocolNotSupported, "Network protocol not supported, use (1,2)")
		return
	}

	host := parts[2]
	port, err := strconv.Atoi(parts[3])
	if err != nil {
		sess.writeMessage(CodeNetworkProtocolNotSupported, "Network protocol not supported, use (1,2)")
		return
	}

	socket, err := newActiveSocket(sess, host, port)
	if err != nil {
		sess.writeMessage(CodeCannotOpenDataConnection, "Data connection failed")
		return
	}

	sess.dataConn = socket
	sess.writeMessage(CodeCommandOK, "Connection established ("+strconv.Itoa(port)+")")
}

// commandLPRT responds to the LPRT FTP command. It allows the client to
// request an active data socket with more options than the original PORT
// command.  FTP Operation Over Big Address Records.
type commandLPRT struct{}

func (cmd commandLPRT) IsExtend() bool {
	return true
}

func (cmd commandLPRT) RequireParam() bool {
	return true
}

func (cmd commandLPRT) RequireAuth() bool {
	return true
}

func (cmd commandLPRT) Execute(sess *Session, param string) {
	// TODO: no tests for this code yet.

	parts := strings.Split(param, ",")

	addressFamily, err := strconv.Atoi(parts[0])
	if err != nil {
		sess.writeMessage(CodeNetworkProtocolNotSupported, "Network protocol not supported, use 4")
		return
	}

	const addressFamilyIPv4 = 4

	if addressFamily != addressFamilyIPv4 {
		sess.writeMessage(CodeNetworkProtocolNotSupported, "Network protocol not supported, use 4")
		return
	}

	addressLength, err := strconv.Atoi(parts[1])
	if err != nil {
		sess.writeMessage(CodeNetworkProtocolNotSupported, "Network protocol not supported, use 4")
		return
	}

	if addressLength != addressFamilyIPv4 {
		sess.writeMessage(CodeNetworkProtocolNotSupported, "Network IP length not supported, use 4")
		return
	}

	host := strings.Join(parts[2:2+addressLength], ".")

	portLength, err := strconv.Atoi(parts[2+addressLength])
	if err != nil {
		sess.writeMessage(CodeNetworkProtocolNotSupported, "Network protocol not supported, use 4")
		return
	}
	portAddress := parts[3+addressLength : 3+addressLength+portLength]

	port, err := parsePortFromBytes(portAddress)
	if err != nil {
		sess.writeMessage(CodeNetworkProtocolNotSupported, "Network protocol not supported, use 4")
		return
	}

	// If the existing connection is on the same, host/port don't reconnect.
	if sess.dataConn.Host() == host && sess.dataConn.Port() == port {
		return
	}

	socket, err := newActiveSocket(sess, host, port)
	if err != nil {
		sess.writeMessage(CodeCannotOpenDataConnection, "Data connection failed")
		return
	}

	sess.dataConn = socket
	sess.writeMessage(CodeCommandOK, "Connection established ("+strconv.Itoa(port)+")")
}

// parsePortFromBytes converts a slice of port address strings to a port number.
// Each string represents a byte of the port number in big-endian order.
func parsePortFromBytes(portAddress []string) (int, error) {
	portBytes := make([]byte, len(portAddress))
	for i, portPart := range portAddress {
		p, err := strconv.Atoi(portPart)
		if err != nil {
			return 0, fmt.Errorf("invalid port byte at position %d: %w", i, err)
		}

		portBytes[i] = byte(p)
	}

	if len(portBytes) != 2 {
		return 0, fmt.Errorf("expected 2 port bytes, got %d", len(portBytes))
	}

	return int(binary.BigEndian.Uint16(portBytes)), nil
}

// commandEPSV responds to the EPSV FTP command. It allows the client to request a passive data socket with more options
// than the original PASV command. It mainly adds IPv6 support, although we don't support that yet.
type commandEPSV struct{}

func (cmd commandEPSV) IsExtend() bool {
	return true
}

func (cmd commandEPSV) RequireParam() bool {
	return false
}

func (cmd commandEPSV) RequireAuth() bool {
	return true
}

func (cmd commandEPSV) Execute(sess *Session, _ string) {
	socket, err := sess.newPassiveSocket()
	if err != nil {
		sess.log(err)
		sess.writeMessage(CodeCannotOpenDataConnection, "Data connection failed")
		return
	}

	msg := fmt.Sprintf("Entering Extended Passive Mode (|||%d|)", socket.Port())
	sess.writeMessage(CodeEnteringExtendedPassiveMode, msg)
}

// commandLIST responds to the LIST FTP command. It allows the client to retrieve a detailed listing of a directory.
type commandLIST struct{}

func (cmd commandLIST) IsExtend() bool {
	return false
}

func (cmd commandLIST) RequireParam() bool {
	return false
}

func (cmd commandLIST) RequireAuth() bool {
	return true
}

func convertFileInfo(sess *Session, f os.FileInfo, p string) (*fileInfo, error) {
	mode, err := sess.server.Perm.GetMode(p)
	if err != nil {
		return nil, fmt.Errorf("get mode: %w", err)
	}
	if f.IsDir() {
		mode |= os.ModeDir
	}

	owner, err := sess.server.Perm.GetOwner(p)
	if err != nil {
		return nil, fmt.Errorf("get owner: %w", err)
	}

	group, err := sess.server.Perm.GetGroup(p)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}

	return &fileInfo{
		FileInfo: f,
		mode:     mode,
		owner:    owner,
		group:    group,
	}, nil
}

func list(sess *Session, cmd, p, param string) ([]FileInfo, error) {
	ctx := &Context{
		Sess:  sess,
		CMD:   cmd,
		Param: param,
		Data:  make(map[string]any),
	}
	info, err := sess.server.Driver.Stat(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}

	if info == nil {
		sess.logf("%s: no such file or directory.\n", p)
		return []FileInfo{}, nil
	}

	var files []FileInfo
	if info.IsDir() {
		if err = sess.server.Driver.ListDir(ctx, p, func(f os.FileInfo) error {
			info, cErr := convertFileInfo(sess, f, path.Join(p, f.Name()))
			if cErr != nil {
				return cErr
			}

			files = append(files, info)

			return nil
		}); err != nil {
			return nil, fmt.Errorf("list dir: %w", err)
		}
	} else {
		newInfo, cErr := convertFileInfo(sess, info, p)
		if cErr != nil {
			return nil, cErr
		}

		files = append(files, newInfo)
	}

	return files, nil
}

func (cmd commandLIST) Execute(sess *Session, param string) {
	p := sess.buildPath(parseListParam(param))

	files, err := list(sess, "LIST", p, param)
	if err != nil {
		sess.writeMessage(CodeFileUnavailable, err.Error())
		return
	}

	sess.writeMessage(CodeFileStatusOK, "Opening ASCII mode data connection for file list")
	sess.sendOutofbandData(listFormatter(files).Detailed())
}

func parseListParam(param string) string {
	if len(param) == 0 {
		return param
	}

	fields := strings.Fields(param)
	i := 0
	for _, field := range fields {
		if !strings.HasPrefix(field, "-") {
			break
		}

		i = strings.LastIndex(param, " "+field) + len(field) + 1
	}

	// Get the full path with whitespace retained.
	return strings.TrimLeft(param[i:], " ")
}

// commandNLST responds to the NLST FTP command. It allows the client to retrieve a list of filenames in the current
// directory.
type commandNLST struct{}

func (cmd commandNLST) IsExtend() bool {
	return false
}

func (cmd commandNLST) RequireParam() bool {
	return false
}

func (cmd commandNLST) RequireAuth() bool {
	return true
}

func (cmd commandNLST) Execute(sess *Session, param string) {
	ctx := &Context{
		Sess:  sess,
		CMD:   "NLST",
		Param: param,
		Data:  make(map[string]any),
	}

	buildPath := sess.buildPath(parseListParam(param))
	info, err := sess.server.Driver.Stat(ctx, buildPath)
	if err != nil {
		sess.writeMessage(CodeFileUnavailable, err.Error())
		return
	}
	if !info.IsDir() {
		sess.writeMessage(CodeFileUnavailable, param+" is not a directory")
		return
	}

	var files []FileInfo
	if err = sess.server.Driver.ListDir(ctx, buildPath, func(f os.FileInfo) error {
		mode, modeErr := sess.server.Perm.GetMode(buildPath)
		if modeErr != nil {
			return fmt.Errorf("get mode: %w", modeErr)
		}

		if info.IsDir() {
			mode |= os.ModeDir
		}

		owner, ownerErr := sess.server.Perm.GetOwner(buildPath)
		if ownerErr != nil {
			return fmt.Errorf("get owner: %w", ownerErr)
		}

		group, groupErr := sess.server.Perm.GetGroup(buildPath)
		if groupErr != nil {
			return fmt.Errorf("get group: %w", groupErr)
		}

		files = append(files, &fileInfo{
			FileInfo: f,
			mode:     mode,
			owner:    owner,
			group:    group,
		})

		return nil
	}); err != nil {
		sess.writeMessage(CodeFileUnavailable, err.Error())
		return
	}

	sess.writeMessage(CodeFileStatusOK, "Opening ASCII mode data connection for file list")
	sess.sendOutofbandData(listFormatter(files).Short())
}

// commandMDTM responds to the MDTM FTP command. It allows the client to retrieve the last modified time of a file.
type commandMDTM struct{}

func (cmd commandMDTM) IsExtend() bool {
	return false
}

func (cmd commandMDTM) RequireParam() bool {
	return true
}

func (cmd commandMDTM) RequireAuth() bool {
	return true
}

func (cmd commandMDTM) Execute(sess *Session, param string) {
	buildPath := sess.buildPath(param)
	stat, err := sess.server.Driver.Stat(&Context{
		Sess:  sess,
		CMD:   "MDTM",
		Param: param,
		Data:  make(map[string]any),
	}, buildPath)
	if err != nil {
		sess.writeMessage(CodeFileActionNotTaken, "File not available")
		return
	}

	sess.writeMessage(CodeFileStatus, stat.ModTime().Format("20060102150405"))
}

// commandMKD responds to the MKD FTP command. It allows the client to create a new directory.
type commandMKD struct{}

func (cmd commandMKD) IsExtend() bool {
	return false
}

func (cmd commandMKD) RequireParam() bool {
	return true
}

func (cmd commandMKD) RequireAuth() bool {
	return true
}

func (cmd commandMKD) Execute(sess *Session, param string) {
	buildPath := sess.buildPath(param)
	ctx := Context{
		Sess:  sess,
		CMD:   "MKD",
		Param: param,
		Data:  make(map[string]any),
	}
	sess.server.notifiers.BeforeCreateDir(&ctx, buildPath)
	err := sess.server.Driver.MakeDir(&ctx, buildPath)
	sess.server.notifiers.AfterDirCreated(&ctx, buildPath, err)
	if err == nil {
		sess.writeMessage(CodePathnameCreated, "Directory created")
	} else {
		sess.writeMessage(CodeFileUnavailable, fmt.Sprint("Action not taken: ", err))
	}
}

// cmdMode responds to the MODE FTP command.
//
// The original FTP spec had various options for hosts to negotiate how data would be sent over the data socket.
// These days (S)tream mode is all that is used for the mode - data is just streamed down the data socket unchanged.
type commandMODE struct{}

func (cmd commandMODE) IsExtend() bool {
	return false
}

func (cmd commandMODE) RequireParam() bool {
	return true
}

func (cmd commandMODE) RequireAuth() bool {
	return true
}

func (cmd commandMODE) Execute(sess *Session, param string) {
	if strings.ToUpper(param) == "S" {
		sess.writeMessage(CodeCommandOK, "OK")
	} else {
		sess.writeMessage(CodeCommandNotImplementedForParm, "MODE is an obsolete command")
	}
}

// cmdNoop responds to the NOOP FTP command.
//
// This is essentially a ping from the client so we just respond with an
// basic 200 message.
type commandNOOP struct{}

func (cmd commandNOOP) IsExtend() bool {
	return false
}

func (cmd commandNOOP) RequireParam() bool {
	return false
}

func (cmd commandNOOP) RequireAuth() bool {
	return false
}

func (cmd commandNOOP) Execute(sess *Session, _ string) {
	sess.writeMessage(CodeCommandOK, "OK")
}

// commandPASS respond to the PASS FTP command by asking the driver if the supplied username and password are valid.
type commandPASS struct{}

func (cmd commandPASS) IsExtend() bool {
	return false
}

func (cmd commandPASS) RequireParam() bool {
	return true
}

func (cmd commandPASS) RequireAuth() bool {
	return false
}

func (cmd commandPASS) Execute(sess *Session, param string) {
	auth := sess.server.Auth

	// If the driver implements Auth, call that instead of the server version.
	if driverAuth, found := sess.server.Driver.(Auth); found {
		auth = driverAuth
	}

	ctx := Context{
		Sess:  sess,
		CMD:   "PASS",
		Param: param,
		Data:  make(map[string]any),
	}

	ok, err := auth.CheckPasswd(&ctx, sess.reqUser, param)
	sess.server.notifiers.AfterUserLogin(&ctx, sess.reqUser, param, ok, err)
	if err != nil {
		sess.writeMessage(CodeFileUnavailable, "Checking password error")
		return
	}

	if ok {
		sess.user = sess.reqUser
		sess.reqUser = ""
		sess.writeMessage(CodeUserLoggedIn, "Password ok, continue")
	} else {
		sess.writeMessage(CodeNotLoggedIn, "Incorrect password, not logged in")
	}
}

// commandPASV responds to the PASV FTP command.
//
// The client is requesting us to open a new TCP listing socket and wait for them
// to connect to it.
type commandPASV struct{}

func (cmd commandPASV) IsExtend() bool {
	return false
}

func (cmd commandPASV) RequireParam() bool {
	return false
}

func (cmd commandPASV) RequireAuth() bool {
	return true
}

func (cmd commandPASV) Execute(sess *Session, _ string) {
	listenIP := sess.passiveListenIP()

	// TODO: IPv6 for this command is not implemented.
	if strings.HasPrefix(listenIP, "::") {
		sess.writeMessage(CodeFileUnavailable, "Action not taken")
		return
	}

	socket, err := sess.newPassiveSocket()
	if err != nil {
		sess.writeMessage(CodeCannotOpenDataConnection, "Data connection failed")
		return
	}

	p1 := socket.Port() / 256
	p2 := socket.Port() - (p1 * 256)

	quads := strings.Split(listenIP, ".")
	msg := fmt.Sprintf("Entering Passive Mode (%s,%s,%s,%s,%d,%d)", quads[0], quads[1], quads[2], quads[3], p1, p2)
	sess.writeMessage(CodeEnteringPassiveMode, msg)
}

// commandPORT responds to the PORT FTP command.
//
// The client has opened a listening socket for sending out of band data and is requesting that we connect to it.
type commandPORT struct{}

func (cmd commandPORT) IsExtend() bool {
	return false
}

func (cmd commandPORT) RequireParam() bool {
	return true
}

func (cmd commandPORT) RequireAuth() bool {
	return true
}

func (cmd commandPORT) Execute(sess *Session, param string) {
	const (
		byteShift     = 256 // Multiplier to convert high byte to port.
		expectedParts = 6   // IP address (4 octets) + port (2 bytes).
		ipOctetCount  = 4   // Number of octets in IP address.
		portHighByte  = 4   // High byte of port number.
		portLowByte   = 5   // Low byte of port number.
	)

	parts := strings.Split(param, ",")
	if len(parts) != expectedParts {
		sess.writeMessage(CodeSyntaxError, "Invalid PORT command format")
		return
	}

	portHigh, err := strconv.Atoi(parts[portHighByte])
	if err != nil {
		sess.writeMessage(CodeSyntaxError, "Invalid port high byte")
		return
	}

	portLow, err := strconv.Atoi(parts[portLowByte])
	if err != nil {
		sess.writeMessage(CodeSyntaxError, "Invalid port low byte")
		return
	}

	port := (portHigh * byteShift) + portLow
	host := strings.Join(parts[:ipOctetCount], ".")

	socket, err := newActiveSocket(sess, host, port)
	if err != nil {
		sess.writeMessage(CodeCannotOpenDataConnection, "Data connection failed")
		return
	}

	sess.dataConn = socket
	sess.writeMessage(CodeCommandOK, fmt.Sprintf("Connection established (%d)", port))
}

// commandPWD responds to the PWD FTP command.
//
// Tells the client what the current working directory is.
type commandPWD struct{}

func (cmd commandPWD) IsExtend() bool {
	return false
}

func (cmd commandPWD) RequireParam() bool {
	return false
}

func (cmd commandPWD) RequireAuth() bool {
	return true
}

func (cmd commandPWD) Execute(sess *Session, _ string) {
	sess.writeMessage(CodePathnameCreated, "\""+sess.curDir+"\" is the current directory")
}

// CommandQuit responds to the QUIT FTP command. The client has requested the
// connection be closed.
type commandQUIT struct{}

func (cmd commandQUIT) IsExtend() bool {
	return false
}

func (cmd commandQUIT) RequireParam() bool {
	return false
}

func (cmd commandQUIT) RequireAuth() bool {
	return false
}

func (cmd commandQUIT) Execute(sess *Session, _ string) {
	sess.writeMessage(CodeServiceClosingControlConn, "Goodbye")

	if err := sess.Close(); err != nil {
		sess.logf("Error closing control connection: %v", err)
	}
}

// commandRETR responds to the RETR FTP command. It allows the client to download a file.
// REST can be followed by APPE, STOR, or RETR.
type commandRETR struct{}

func (cmd commandRETR) IsExtend() bool {
	return false
}

func (cmd commandRETR) RequireParam() bool {
	return true
}

func (cmd commandRETR) RequireAuth() bool {
	return true
}

func (cmd commandRETR) Execute(sess *Session, param string) {
	buildPath := sess.buildPath(param)
	if sess.preCommand != "REST" {
		sess.lastFilePos = -1
	}

	defer func() {
		sess.lastFilePos = -1
	}()

	ctx := Context{
		Sess:  sess,
		CMD:   "RETR",
		Param: param,
		Data:  make(map[string]any),
	}

	sess.server.notifiers.BeforeDownloadFile(&ctx, buildPath)
	readPos := sess.lastFilePos
	if readPos < 0 {
		readPos = 0
	}

	size, data, err := sess.server.Driver.GetFile(&ctx, buildPath, readPos)
	if err != nil {
		sess.server.notifiers.AfterFileDownloaded(&ctx, buildPath, size, err)
		sess.writeMessage(CodeActionAbortedPageUnknown, "File not available")
		return
	}
	defer data.Close()

	sess.writeMessage(CodeFileStatusOK, fmt.Sprintf("Data transfer starting %d bytes", size))
	err = sess.sendOutOfBandDataWriter(data)
	sess.server.notifiers.AfterFileDownloaded(&ctx, buildPath, size, err)
	if err != nil {
		sess.writeMessage(CodeActionAbortedPageUnknown, "Error reading file")
	}
}

type commandREST struct{}

func (cmd commandREST) IsExtend() bool {
	return false
}

func (cmd commandREST) RequireParam() bool {
	return true
}

func (cmd commandREST) RequireAuth() bool {
	return true
}

func (cmd commandREST) Execute(sess *Session, param string) {
	var err error
	sess.lastFilePos, err = strconv.ParseInt(param, 10, 64)
	if err != nil {
		sess.writeMessage(CodeActionAbortedPageUnknown, "File not available")
		return
	}

	sess.writeMessage(CodeRequestedFileActionPending, fmt.Sprint("Start transfer from ", sess.lastFilePos))
}

// commandRNFR responds to the RNFR FTP command. It's the first of two commands
// required for a client to rename a file.
type commandRNFR struct{}

func (cmd commandRNFR) IsExtend() bool {
	return false
}

func (cmd commandRNFR) RequireParam() bool {
	return true
}

func (cmd commandRNFR) RequireAuth() bool {
	return true
}

func (cmd commandRNFR) Execute(sess *Session, param string) {
	sess.renameFrom = ""
	p := sess.buildPath(param)
	if _, err := sess.server.Driver.Stat(&Context{
		Sess:  sess,
		CMD:   "RNFR",
		Param: param,
		Data:  make(map[string]any),
	}, p); err != nil {
		sess.writeMessage(CodeFileUnavailable, fmt.Sprint("Action not taken: ", err))
		return
	}

	sess.renameFrom = p
	sess.writeMessage(CodeRequestedFileActionPending, "Requested file action pending further information.")
}

// cmdRnto responds to the RNTO FTP command. It's the second of two commands
// required for a client to rename a file.
type commandRNTO struct{}

func (cmd commandRNTO) IsExtend() bool {
	return false
}

func (cmd commandRNTO) RequireParam() bool {
	return true
}

func (cmd commandRNTO) RequireAuth() bool {
	return true
}

func (cmd commandRNTO) Execute(sess *Session, param string) {
	toPath := sess.buildPath(param)
	err := sess.server.Driver.Rename(&Context{
		Sess:  sess,
		CMD:   "RNTO",
		Param: param,
		Data:  make(map[string]any),
	}, sess.renameFrom, toPath)
	defer func() {
		sess.renameFrom = ""
	}()

	if err == nil {
		sess.writeMessage(CodeFileActionOK, "File renamed")
	} else {
		sess.writeMessage(CodeFileUnavailable, fmt.Sprint("Action not taken: ", err))
	}
}

// cmdRmd responds to the RMD FTP command. It allows the client to delete a directory.
type commandRMD struct{}

func (cmd commandRMD) IsExtend() bool {
	return false
}

func (cmd commandRMD) RequireParam() bool {
	return true
}

func (cmd commandRMD) RequireAuth() bool {
	return true
}

func (cmd commandRMD) Execute(sess *Session, param string) {
	executeRmd("RMD", sess, param)
}

// cmdXRmd responds to the RMD FTP command. It allows the client to delete a directory.
type commandXRmd struct{}

func (cmd commandXRmd) IsExtend() bool {
	return false
}

func (cmd commandXRmd) RequireParam() bool {
	return true
}

func (cmd commandXRmd) RequireAuth() bool {
	return true
}

func (cmd commandXRmd) Execute(sess *Session, param string) {
	executeRmd("XRMD", sess, param)
}

func executeRmd(cmd string, sess *Session, param string) {
	ctx := newContext(sess, cmd, param, nil)
	if param == "/" || param == "" {
		sess.writeMessage(CodeFileUnavailable, "Directory / cannot be deleted")
		return
	}

	realPath := sess.buildPath(param)
	needChangeCurDir := strings.HasPrefix(param, sess.curDir)

	sess.server.notifiers.BeforeDeleteDir(ctx, realPath)
	err := sess.server.Driver.DeleteDir(ctx, realPath)
	sess.server.notifiers.AfterDirDeleted(ctx, realPath, err)
	if needChangeCurDir {
		if changeDirErr := sess.changeCurDir(path.Dir(param)); changeDirErr != nil && err == nil {
			err = fmt.Errorf("change directory: %w", changeDirErr)
		}
	}
	if err != nil {
		sess.writeMessage(CodeFileUnavailable, fmt.Sprintf("Directory delete failed: %v", err))
		return

	}

	sess.writeMessage(CodeFileActionOK, "Directory deleted")
}

type commandADAT struct{}

func (cmd commandADAT) IsExtend() bool {
	return false
}

func (cmd commandADAT) RequireParam() bool {
	return true
}

func (cmd commandADAT) RequireAuth() bool {
	return true
}

func (cmd commandADAT) Execute(sess *Session, _ string) {
	sess.writeMessage(CodeFileUnavailable, "Action not taken")
}

type commandAUTH struct{}

func (cmd commandAUTH) IsExtend() bool {
	return false
}

func (cmd commandAUTH) RequireParam() bool {
	return true
}

func (cmd commandAUTH) RequireAuth() bool {
	return false
}

func (cmd commandAUTH) Execute(sess *Session, param string) {
	if param == "TLS" && sess.server.TLSConfig != nil {
		sess.writeMessage(CodeSecurityMechanismAccepted, "AUTH command OK")

		if err := sess.upgradeToTLS(); err != nil {
			sess.logf("Error upgrading connection to TLS %v", err.Error())
		}
	} else {
		sess.writeMessage(CodeFileUnavailable, "Action not taken")
	}
}

type commandCCC struct{}

func (cmd commandCCC) IsExtend() bool {
	return false
}

func (cmd commandCCC) RequireParam() bool {
	return true
}

func (cmd commandCCC) RequireAuth() bool {
	return true
}

func (cmd commandCCC) Execute(sess *Session, _ string) {
	sess.writeMessage(CodeFileUnavailable, "Action not taken")
}

type commandENC struct{}

func (cmd commandENC) IsExtend() bool {
	return false
}

func (cmd commandENC) RequireParam() bool {
	return true
}

func (cmd commandENC) RequireAuth() bool {
	return true
}

func (cmd commandENC) Execute(sess *Session, _ string) {
	sess.writeMessage(CodeFileUnavailable, "Action not taken")
}

type commandMIC struct{}

func (cmd commandMIC) IsExtend() bool {
	return false
}

func (cmd commandMIC) RequireParam() bool {
	return true
}

func (cmd commandMIC) RequireAuth() bool {
	return true
}

func (cmd commandMIC) Execute(sess *Session, _ string) {
	sess.writeMessage(CodeFileUnavailable, "Action not taken")
}

type commandMLSD struct{}

func (cmd commandMLSD) IsExtend() bool {
	return true
}

func (cmd commandMLSD) RequireParam() bool {
	return false
}

func (cmd commandMLSD) RequireAuth() bool {
	return true
}

func toMLSDFormat(files []FileInfo) []byte {
	var buf bytes.Buffer
	for _, file := range files {
		fileType := "file"
		if file.IsDir() {
			fileType = "dir"
		}
		/*Possible facts "Size" / "Modify" / "Create" /
				  "Type" / "Unique" / "Perm" /
				  "Lang" / "Media-Type" / "CharSet"
				  TODO: Perm pvals        = "a" / "c" / "d" / "e" / "f" /
		                     "l" / "m" / "p" / "r" / "w"
		*/
		fmt.Fprintf(&buf,
			"Type=%s;Modify=%s;Size=%d; %s\n",
			fileType,
			file.ModTime().Format("20060102150405"),
			file.Size(),
			file.Name(),
		)
	}
	return buf.Bytes()
}

func (cmd commandMLSD) Execute(sess *Session, param string) {
	if param == "" {
		param = sess.curDir
	}
	p := sess.buildPath(param)

	files, err := list(sess, "MLSD", p, param)
	if err != nil {
		sess.writeMessage(CodeFileUnavailable, err.Error())
		return
	}

	sess.writeMessage(CodeFileStatusOK, "Opening ASCII mode data connection for file list")
	sess.sendOutofbandData(toMLSDFormat(files))
}

type commandPBSZ struct{}

func (cmd commandPBSZ) IsExtend() bool {
	return false
}

func (cmd commandPBSZ) RequireParam() bool {
	return true
}

func (cmd commandPBSZ) RequireAuth() bool {
	return false
}

func (cmd commandPBSZ) Execute(sess *Session, param string) {
	if sess.tls && param == "0" {
		sess.writeMessage(CodeCommandOK, "OK")
	} else {
		sess.writeMessage(CodeFileUnavailable, "Action not taken")
	}
}

const (
	ProtectionLevelClear   = "C" // ProtectionLevelClear represents clear (unencrypted) mode.
	ProtectionLevelPrivate = "P" // ProtectionLevelPrivate represents private (encrypted) mode.
)

type commandPROT struct{}

func (cmd commandPROT) IsExtend() bool {
	return false
}

func (cmd commandPROT) RequireParam() bool {
	return true
}

func (cmd commandPROT) RequireAuth() bool {
	return false
}

func (cmd commandPROT) Execute(sess *Session, param string) {
	if sess.tls {
		if param == ProtectionLevelPrivate {
			sess.writeMessage(CodeCommandOK, "OK")
			return
		}

		sess.writeMessage(CodeDataProtectionNotSupported, "Only P level is supported")
		return
	}

	sess.writeMessage(CodeFileUnavailable, "Action not taken")
}

type commandCONF struct{}

func (cmd commandCONF) IsExtend() bool {
	return false
}

func (cmd commandCONF) RequireParam() bool {
	return true
}

func (cmd commandCONF) RequireAuth() bool {
	return true
}

func (cmd commandCONF) Execute(sess *Session, _ string) {
	sess.writeMessage(CodeFileUnavailable, "Action not taken")
}

// commandSIZE responds to the SIZE FTP command. It returns the size of the
// requested path in bytes.
type commandSIZE struct{}

func (cmd commandSIZE) IsExtend() bool {
	return false
}

func (cmd commandSIZE) RequireParam() bool {
	return true
}

func (cmd commandSIZE) RequireAuth() bool {
	return true
}

func (cmd commandSIZE) Execute(sess *Session, param string) {
	buildPath := sess.buildPath(param)
	stat, err := sess.server.Driver.Stat(&Context{
		Sess:  sess,
		CMD:   "SIZE",
		Param: param,
		Data:  make(map[string]any),
	}, buildPath)
	if err != nil {
		// TODO: shouldn't log without using upstream logger.
		log.Printf("Size: error(%s)", err)
		sess.writeMessage(CodeFileActionNotTaken, fmt.Sprintf("path %s not found", param))
	} else {
		sess.writeMessage(CodeFileStatus, strconv.Itoa(int(stat.Size())))
	}
}

// commandSTAT responds to the STAT FTP command. It returns the stat of the requested path.
type commandSTAT struct{}

func (cmd commandSTAT) IsExtend() bool {
	return false
}

func (cmd commandSTAT) RequireParam() bool {
	return false
}

func (cmd commandSTAT) RequireAuth() bool {
	return true
}

func (cmd commandSTAT) Execute(sess *Session, param string) {
	// System stat.
	if param == "" {
		sess.writeMessage(CodeSystemStatus, fmt.Sprintf("%s FTP server status:\nVersion %s"+
			"Connected to %s (%s)\n"+
			"Logged in %s\n"+
			"TYPE: ASCII, FORM: Nonprint; STRUcture: File; transfer MODE: Stream\n"+
			"No data connection", sess.PublicIP(), version, sess.PublicIP(),
			version, sess.LoginUser()))
		sess.writeMessage(CodeSystemStatus, "End of status")
		return
	}

	ctx := Context{
		Sess:  sess,
		CMD:   "STAT",
		Param: param,
		Data:  make(map[string]any),
	}

	// File or directory stat.
	buildPath := sess.buildPath(param)

	stat, err := sess.server.Driver.Stat(&ctx, buildPath)
	if err != nil {
		log.Printf("Size: error(%s)", err)
		sess.writeMessage(CodeFileActionNotTaken, fmt.Sprintf("path %s not found", buildPath))
	} else {
		var files []FileInfo

		if stat.IsDir() {
			err = sess.server.Driver.ListDir(&ctx, buildPath, func(f os.FileInfo) error {
				info, cErr := convertFileInfo(sess, f, filepath.Join(buildPath, f.Name()))
				if cErr != nil {
					return cErr
				}

				files = append(files, info)

				return nil
			})
			if err != nil {
				sess.writeMessage(CodeFileUnavailable, err.Error())
				return
			}

			sess.writeMessage(CodeFileStatus, "Opening ASCII mode data connection for file list")
		} else {
			info, cErr := convertFileInfo(sess, stat, buildPath)
			if cErr != nil {
				sess.writeMessage(CodeFileUnavailable, cErr.Error())
				return
			}

			files = append(files, info)
			sess.writeMessage(CodeDirectoryStatus, "Opening ASCII mode data connection for file list")
		}
		sess.sendOutofbandData(listFormatter(files).Detailed())
	}
}

// commandSTOR responds to the STOR FTP command. It allows the user to upload a new file.
type commandSTOR struct{}

func (cmd commandSTOR) IsExtend() bool {
	return false
}

func (cmd commandSTOR) RequireParam() bool {
	return true
}

func (cmd commandSTOR) RequireAuth() bool {
	return true
}

func (cmd commandSTOR) Execute(sess *Session, param string) {
	targetPath := sess.buildPath(param)
	sess.writeMessage(CodeFileStatusOK, "Data transfer starting")

	if sess.preCommand != "REST" {
		sess.lastFilePos = -1
	}

	defer func() {
		sess.lastFilePos = -1
	}()

	ctx := Context{
		Sess:  sess,
		CMD:   "STOR",
		Param: param,
		Data:  make(map[string]any),
	}
	sess.server.notifiers.BeforePutFile(&ctx, targetPath)
	size, err := sess.server.Driver.PutFile(&ctx, targetPath, sess.dataConn, sess.lastFilePos)
	sess.server.notifiers.AfterFilePut(&ctx, targetPath, size, err)
	if err != nil {
		sess.writeMessage(CodeFileActionNotTaken, fmt.Sprint("error during transfer: ", err))
		return
	}

	sess.writeMessage(CodeClosingDataConnection, fmt.Sprintf("OK, received %d bytes", size))
}

// commandSTRU handles the STRU (structure) FTP command.
//
// Like the MODE and TYPE commands, STRU originates from an earlier era of FTP when the protocol was aware of file
// structure and might transform data (e.g., end-of-line markers) during transfer.
//
// Modern FTP servers transmit files as raw bytes, so only File (F) structure mode is relevant and supported.
type commandSTRU struct{}

func (cmd commandSTRU) IsExtend() bool {
	return false
}

func (cmd commandSTRU) RequireParam() bool {
	return true
}

func (cmd commandSTRU) RequireAuth() bool {
	return true
}

func (cmd commandSTRU) Execute(sess *Session, param string) {
	if strings.ToUpper(param) == "F" {
		sess.writeMessage(CodeCommandOK, "OK")
	} else {
		sess.writeMessage(CodeCommandNotImplementedForParm, "STRU is an obsolete command")
	}
}

// commandSYST responds to the SYST FTP command by providing a canned response.
type commandSYST struct{}

func (cmd commandSYST) IsExtend() bool {
	return false
}

func (cmd commandSYST) RequireParam() bool {
	return false
}

func (cmd commandSYST) RequireAuth() bool {
	return true
}

func (cmd commandSYST) Execute(sess *Session, _ string) {
	sess.writeMessage(CodeSystemType, "UNIX Type: L8")
}

// commandTYPE handles the TYPE FTP command.
//
// Historically, FTP supported different transfer modes (TYPE, MODE, and STRU) from a time when the protocol was aware
// of file content and might translate things like end-of-line markers during transfer.
//
// Valid TYPE arguments include A(SCII), I(mage), E(BCDIC), and L(N) for local byte size. Since this server transfers
// raw bytes without translation, Image mode is enough. However, the RFC requires support for ASCII mode as well, so
// we accept it but treat it identically to Image mode.
type commandTYPE struct{}

func (cmd commandTYPE) IsExtend() bool {
	return false
}

func (cmd commandTYPE) RequireParam() bool {
	return false
}

func (cmd commandTYPE) RequireAuth() bool {
	return true
}

func (cmd commandTYPE) Execute(sess *Session, param string) {
	param = strings.ToUpper(param)

	switch param {
	case "A":
		sess.writeMessage(CodeCommandOK, "Type set to ASCII")
		return
	case "I":
		sess.writeMessage(CodeCommandOK, "Type set to binary")
		return
	}

	sess.writeMessage(CodeSyntaxError, "Invalid type")
}

// commandUSER responds to the USER FTP command by asking for the password.
type commandUSER struct{}

func (cmd commandUSER) IsExtend() bool {
	return false
}

func (cmd commandUSER) RequireParam() bool {
	return true
}

func (cmd commandUSER) RequireAuth() bool {
	return false
}

func (cmd commandUSER) Execute(sess *Session, param string) {
	sess.reqUser = param
	sess.server.notifiers.BeforeLoginUser(&Context{
		Sess:  sess,
		CMD:   "USER",
		Param: param,
		Data:  make(map[string]any),
	}, sess.reqUser)
	sess.writeMessage(CodeUserNameOKNeedPassword, "User name ok, password required")
}
