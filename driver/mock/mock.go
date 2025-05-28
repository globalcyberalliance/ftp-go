// Copyright 2020 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mock

import (
	"io"
	"os"

	"github.com/globalcyberalliance/ftp-go"
)

// Driver implements Driver directly read local file system.
type Driver struct{}

// NewDriver implements Driver.
func NewDriver() (ftp.Driver, error) {
	return &Driver{}, nil
}

// Stat implements Driver.
func (driver *Driver) Stat(ctx *ftp.Context, path string) (os.FileInfo, error) {
	return nil, nil
}

// ListDir implements Driver.
func (driver *Driver) ListDir(ctx *ftp.Context, path string, callback func(os.FileInfo) error) error {
	return nil
}

// DeleteDir implements Driver.
func (driver *Driver) DeleteDir(ctx *ftp.Context, path string) error {
	return nil
}

// DeleteFile implements Driver.
func (driver *Driver) DeleteFile(ctx *ftp.Context, path string) error {
	return nil
}

// Rename implements Driver.
func (driver *Driver) Rename(ctx *ftp.Context, fromPath string, toPath string) error {
	return nil
}

// MakeDir implements Driver.
func (driver *Driver) MakeDir(ctx *ftp.Context, path string) error {
	return nil
}

// GetFile implements Driver.
func (driver *Driver) GetFile(ctx *ftp.Context, path string, offset int64) (int64, io.ReadCloser, error) {
	return 0, nil, nil
}

// PutFile implements Driver.
func (driver *Driver) PutFile(ctx *ftp.Context, destPath string, data io.Reader, offset int64) (int64, error) {
	return 0, nil
}
