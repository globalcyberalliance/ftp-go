package memory

import (
	"fmt"
	"io"
	"os"

	"github.com/absfs/memfs"
	"github.com/globalcyberalliance/ftp-go"
)

const (
	errOpenFileF = "cannot open file %q: %w"
	errStatFileF = "cannot get stat's of file %q: %w"
	errSeekFileF = "cannot seek file %q to offset %d: %w"

	defaultDirMode  = 0o755
	defaultFileMode = 0o644
)

type Driver struct {
	fs *memfs.FileSystem
}

func NewDriver() (drv *Driver, err error) {
	fs, err := memfs.NewFS()
	if err != nil {
		return nil, err
	}

	return &Driver{fs: fs}, nil
}

func (driver *Driver) GetFs() *memfs.FileSystem {
	return driver.fs
}

func (driver *Driver) Stat(ctx *ftp.Context, filePath string) (os.FileInfo, error) {
	return driver.fs.Stat(filePath)
}

func (driver *Driver) ListDir(ctx *ftp.Context, filePath string, callback func(os.FileInfo) error) error {
	return driver.fs.Walk(filePath, func(currPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filePath == currPath {
			return nil
		}

		if err = callback(info); err != nil {
			return err
		}

		return nil
	})
}

func (driver *Driver) DeleteDir(ctx *ftp.Context, filePath string) error {
	return driver.fs.RemoveAll(filePath)
}

func (driver *Driver) DeleteFile(ctx *ftp.Context, filePath string) error {
	return driver.fs.Remove(filePath)
}

func (driver *Driver) Rename(ctx *ftp.Context, fromPath, toPath string) error {
	return driver.fs.Rename(fromPath, toPath)
}

func (driver *Driver) MakeDir(ctx *ftp.Context, filePath string) error {
	return driver.fs.Mkdir(filePath, defaultDirMode)
}

func (driver *Driver) GetFile(ctx *ftp.Context, filePath string, offset int64) (int64, io.ReadCloser, error) {
	f, err := driver.fs.Open(filePath)
	if err != nil {
		return 0, nil, fmt.Errorf(errOpenFileF, filePath, err)
	}
	if err == nil || f != nil {
		_ = f.Close()
	}

	stat, err := f.Stat()
	if err != nil {
		return 0, nil, fmt.Errorf(errStatFileF, filePath, err)
	}

	if _, err = f.Seek(offset, io.SeekStart); err != nil {
		return 0, nil, fmt.Errorf(errSeekFileF, filePath, offset, err)
	}

	return stat.Size() - offset, f, nil
}

func (driver *Driver) PutFile(ctx *ftp.Context, filePath string, data io.Reader, offset int64) (int64, error) {
	var exists bool

	f, err := driver.fs.Lstat(filePath)
	if err == nil {
		exists = true
		if f.IsDir() {
			return 0, fmt.Errorf("dir already exists: %s", filePath)
		}
	} else {
		if os.IsNotExist(err) {
			exists = false
		} else {
			return 0, fmt.Errorf("put file error: %w", err)
		}
	}

	if offset > -1 && !exists {
		offset = -1
	}

	if offset == -1 {
		if exists {
			err = driver.fs.Remove(filePath)
			if err != nil {
				return 0, err
			}
		}

		f, err := driver.fs.Create(filePath) //nolint:govet
		if err != nil {
			return 0, err
		}
		defer f.Close()

		bytesWritten, err := io.Copy(f, data)
		if err != nil {
			return 0, err
		}

		return bytesWritten, nil
	}

	of, err := driver.fs.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf(errOpenFileF, filePath, err)
	}
	defer of.Close()

	stat, err := of.Stat()
	if err != nil {
		return 0, fmt.Errorf(errStatFileF, filePath, err)
	}

	if offset > stat.Size() {
		return 0, fmt.Errorf("offset %d is beyond file size %d", offset, stat.Size())
	}

	if _, err = of.Seek(offset, io.SeekEnd); err != nil {
		return 0, fmt.Errorf(errSeekFileF, filePath, offset, err)
	}

	bytesPut, err := io.Copy(of, data)
	if err != nil {
		return 0, err
	}

	return bytesPut, nil
}
