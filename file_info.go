// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp

import "os"

type fileInfo struct {
	os.FileInfo
	owner string
	group string
	mode  os.FileMode
}

func (f *fileInfo) Mode() os.FileMode {
	return f.mode
}

func (f *fileInfo) Owner() string {
	return f.owner
}

func (f *fileInfo) Group() string {
	return f.group
}
