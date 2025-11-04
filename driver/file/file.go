// Copyright 2020 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package file

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/globalcyberalliance/ftp-go"
)

// Driver represents a file system driver with a specified root path.
// RootPath defines the base directory for file operations.
type Driver struct {
	RootPath string
}

// NewDriver implements Driver.
func NewDriver(rootPath string) (ftp.Driver, error) {
	absolutePath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("get absolute path: %w", err)
	}

	return &Driver{absolutePath}, nil
}

func (driver *Driver) realPath(path string) string {
	paths := strings.Split(path, "/")
	return filepath.Join(append([]string{driver.RootPath}, paths...)...)
}

// Stat implements Driver.
func (driver *Driver) Stat(_ *ftp.Context, path string) (os.FileInfo, error) {
	basepath := driver.realPath(path)
	rPath, err := filepath.Abs(basepath)
	if err != nil {
		return nil, fmt.Errorf("get absolute path: %w", err)
	}

	fInfo, statErr := os.Lstat(rPath)
	if statErr != nil {
		return nil, fmt.Errorf("stat %w", statErr)
	}

	return fInfo, nil
}

// ListDir implements Driver.
func (driver *Driver) ListDir(_ *ftp.Context, path string, callback func(os.FileInfo) error) error {
	basepath := driver.realPath(path)

	if err := filepath.Walk(basepath, func(f string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rPath, _ := filepath.Rel(basepath, f)
		if rPath == info.Name() {
			if err = callback(info); err != nil {
				return fmt.Errorf("callback: %w", err)
			}

			if info.IsDir() {
				return filepath.SkipDir
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walk dir: %w", err)
	}

	return nil
}

// DeleteDir implements Driver.
func (driver *Driver) DeleteDir(_ *ftp.Context, path string) error {
	rPath := driver.realPath(path)
	f, err := os.Lstat(rPath)
	if err != nil {
		return fmt.Errorf("stat dir: %w", err)
	}

	if f.IsDir() {
		if err = os.Remove(rPath); err != nil {
			return fmt.Errorf("remove dir: %w", err)
		}
	}

	return errors.New("not a directory")
}

// DeleteFile implements Driver.
func (driver *Driver) DeleteFile(_ *ftp.Context, path string) error {
	rPath := driver.realPath(path)
	f, err := os.Lstat(rPath)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	if !f.IsDir() {
		if err = os.Remove(rPath); err != nil {
			return fmt.Errorf("remove file: %w", err)
		}
	}

	return errors.New("not a file")
}

// Rename implements Driver.
func (driver *Driver) Rename(_ *ftp.Context, fromPath string, toPath string) error {
	oldPath := driver.realPath(fromPath)
	newPath := driver.realPath(toPath)

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("rename file: %w", err)
	}

	return nil
}

// MakeDir implements Driver.
func (driver *Driver) MakeDir(_ *ftp.Context, path string) error {
	rPath := driver.realPath(path)

	if err := os.MkdirAll(rPath, os.ModePerm); err != nil {
		return fmt.Errorf("make dir: %w", err)
	}

	return nil
}

// GetFile implements Driver.
func (driver *Driver) GetFile(_ *ftp.Context, path string, offset int64) (int64, io.ReadCloser, error) {
	rPath := driver.realPath(path)
	f, err := os.Open(rPath)
	if err != nil {
		return 0, nil, fmt.Errorf("open file: %w", err)
	}
	defer func() {
		if err != nil && f != nil {
			if closeErr := f.Close(); closeErr != nil {
				err = fmt.Errorf("close file: %w; original error: %w", closeErr, err)
			}
		}
	}()

	info, err := f.Stat()
	if err != nil {
		return 0, nil, fmt.Errorf("stat file: %w", err)
	}

	if _, err = f.Seek(offset, io.SeekStart); err != nil {
		return 0, nil, fmt.Errorf("seek file: %w", err)
	}

	return info.Size() - offset, f, nil
}

// PutFile implements Driver.
func (driver *Driver) PutFile(_ *ftp.Context, destPath string, data io.Reader, offset int64) (int64, error) {
	rPath := driver.realPath(destPath)
	var exists bool

	f, err := os.Lstat(rPath)
	if err != nil {
		exists = false
		if !os.IsNotExist(err) {
			return 0, fmt.Errorf("put file: %w", err)
		}
	} else {
		exists = true
		if f.IsDir() {
			return 0, fmt.Errorf("dir already exists: %s", rPath)
		}
	}

	if offset > -1 && !exists {
		offset = -1
	}

	if offset == -1 {
		if exists {
			if err = os.Remove(rPath); err != nil {
				return 0, fmt.Errorf("remove file: %w", err)
			}
		}

		f, createErr := os.Create(rPath)
		if createErr != nil {
			return 0, fmt.Errorf("create file: %w", createErr)
		}
		defer f.Close()

		bytes, copyErr := io.Copy(f, data)
		if copyErr != nil {
			return 0, fmt.Errorf("write bytes: %w", copyErr)
		}

		return bytes, nil
	}

	of, err := os.OpenFile(rPath, os.O_APPEND|os.O_RDWR, 0o660)
	if err != nil {
		return 0, fmt.Errorf("open file: %w", err)
	}
	defer of.Close()

	info, err := of.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat file: %w", err)
	}

	if offset > info.Size() {
		return 0, fmt.Errorf("offset %d is beyond file size %d", offset, info.Size())
	}

	if _, err = of.Seek(offset, io.SeekEnd); err != nil {
		return 0, fmt.Errorf("seek end: %w", err)
	}

	bytes, err := io.Copy(of, data)
	if err != nil {
		return 0, fmt.Errorf("write bytes: %w", err)
	}

	return bytes, nil
}
