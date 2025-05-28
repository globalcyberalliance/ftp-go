// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp

import (
	"crypto/subtle"
	"regexp"
)

// Auth is an interface to auth your ftp user login.
type Auth interface {
	CheckPasswd(*Context, string, string) (bool, error)
}

var (
	_ Auth = &SimpleAuth{}
	_ Auth = &RegexAuth{}
)

// SimpleAuth implements Auth interface to provide a memory user login auth.
type SimpleAuth struct {
	Name     string
	Password string
}

// CheckPasswd will check user's password.
func (a *SimpleAuth) CheckPasswd(ctx *Context, name, pass string) (bool, error) {
	return constantTimeEquals(name, a.Name) && constantTimeEquals(pass, a.Password), nil
}

func constantTimeEquals(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// RegexAuth implements Auth interface to provide a memory user login auth.
type RegexAuth struct {
	passwordRegex *regexp.Regexp
	usernameRegex *regexp.Regexp
}

func NewRegexAuth(passwordRegex string, usernameRegex string) *RegexAuth {
	return &RegexAuth{
		passwordRegex: regexp.MustCompile(passwordRegex),
		usernameRegex: regexp.MustCompile(usernameRegex),
	}
}

// CheckPasswd will check user's password.
func (a *RegexAuth) CheckPasswd(ctx *Context, username, pass string) (bool, error) {
	if a.passwordRegex.MatchString(pass) && a.usernameRegex.MatchString(username) {
		return true, nil
	}

	return false, nil
}
