// Copyright 2020 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp

// Context represents a context the driver may want to know
type Context struct {
	Sess  *Session
	Data  map[string]interface{} // share data between middlewares
	Cmd   string                 // request command on this request
	Param string                 // request param on this request
}
