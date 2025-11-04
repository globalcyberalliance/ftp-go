// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp

import (
	"fmt"
	"log"
)

// Logger represents an interface to record all ftp information and command.
type Logger interface {
	Print(sessionID string, message any)
	Printf(sessionID string, format string, v ...any)
	PrintCommand(sessionID string, command string, params string)
	PrintResponse(sessionID string, code int, message string)
}

// StdLogger use an instance of this to log in a standard format.
type StdLogger struct{}

// Print implements Logger.
func (logger *StdLogger) Print(sessionID string, message any) {
	log.Printf("%s  %s", sessionID, message)
}

// Printf implements Logger.
func (logger *StdLogger) Printf(sessionID string, format string, v ...any) {
	logger.Print(sessionID, fmt.Sprintf(format, v...))
}

// PrintCommand implements Logger.
func (logger *StdLogger) PrintCommand(sessionID string, command string, params string) {
	if command == "PASS" {
		log.Printf("%s > PASS ****", sessionID)
	} else {
		log.Printf("%s > %s %s", sessionID, command, params)
	}
}

// PrintResponse implements Logger.
func (logger *StdLogger) PrintResponse(sessionID string, code int, message string) {
	log.Printf("%s < %d %s", sessionID, code, message)
}

// DiscardLogger represents a silent logger, produces no output.
type DiscardLogger struct{}

// Print implements Logger.
func (logger *DiscardLogger) Print(_ string, _ any) {}

// Printf implements Logger.
func (logger *DiscardLogger) Printf(_ string, _ string, _ ...any) {}

// PrintCommand implements Logger.
func (logger *DiscardLogger) PrintCommand(_ string, _ string, _ string) {}

// PrintResponse implements Logger.
func (logger *DiscardLogger) PrintResponse(_ string, _ int, _ string) {}
