// Copyright 2020 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mock

import (
	"io"
	"os"

	"github.com/globalcyberalliance/ftp-go"
)

// Driver represents an implementation of the ftp.Driver interface with various file system operations.
type Driver struct{}

// NewDriver implements Driver.
func NewDriver() (ftp.Driver, error) {
	return &Driver{}, nil
}

// Stat implements Driver.
func (driver *Driver) Stat(_ *ftp.Context, _ string) (os.FileInfo, error) {
	return nil, nil
}

// ListDir implements Driver.
func (driver *Driver) ListDir(_ *ftp.Context, _ string, _ func(os.FileInfo) error) error {
	return nil
}

// DeleteDir implements Driver.
func (driver *Driver) DeleteDir(_ *ftp.Context, _ string) error {
	return nil
}

// DeleteFile implements Driver.
func (driver *Driver) DeleteFile(_ *ftp.Context, _ string) error {
	return nil
}

// Rename implements Driver.
func (driver *Driver) Rename(_ *ftp.Context, _ string, _ string) error {
	return nil
}

// MakeDir implements Driver.
func (driver *Driver) MakeDir(_ *ftp.Context, _ string) error {
	return nil
}

// GetFile implements Driver.
func (driver *Driver) GetFile(_ *ftp.Context, _ string, _ int64) (int64, io.ReadCloser, error) {
	return 0, nil, nil
}

// PutFile implements Driver.
func (driver *Driver) PutFile(_ *ftp.Context, _ string, _ io.Reader, _ int64) (int64, error) {
	return 0, nil
}
