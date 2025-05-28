// Copyright 2020 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package integrations

import (
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/globalcyberalliance/ftp-go"
	"github.com/globalcyberalliance/ftp-go/driver/file"
	ftpCli "github.com/jlaffaye/ftp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileDriver(t *testing.T) {
	err := os.MkdirAll("./testdata", os.ModePerm)
	require.NoError(t, err)

	perm := ftp.NewSimplePerm("test", "test")
	driver, err := file.NewDriver("./testdata")
	require.NoError(t, err)

	opt := &ftp.Options{
		Name:   "test ftpd",
		Driver: driver,
		Perm:   perm,
		Port:   2122,
		Auth: &ftp.SimpleAuth{
			Name:     "admin",
			Password: "admin",
		},
		Logger: new(ftp.DiscardLogger),
	}

	runServer(t, opt, nil, func() {
		// Give server 0.5 seconds to get to the listening state
		timeout := time.NewTimer(time.Millisecond * 500)

		for {
			f, err := ftpCli.Connect("localhost:2122")
			if err != nil && len(timeout.C) == 0 { // Retry errors
				continue
			}
			require.NoError(t, err)

			require.NoError(t, f.Login("admin", "admin"))
			assert.Error(t, f.Login("admin", ""))

			content := `test`
			require.NoError(t, f.Stor("server_test.go", strings.NewReader(content)))

			names, err := f.NameList("/")
			require.NoError(t, err)
			assert.Len(t, names, 1)
			assert.EqualValues(t, "server_test.go", names[0])

			bs, err := ioutil.ReadFile("./testdata/server_test.go")
			require.NoError(t, err)
			assert.EqualValues(t, content, string(bs))

			entries, err := f.List("/")
			require.NoError(t, err)
			assert.Len(t, entries, 1)
			assert.EqualValues(t, "server_test.go", entries[0].Name)
			assert.EqualValues(t, 4, entries[0].Size)
			assert.EqualValues(t, ftpCli.EntryTypeFile, entries[0].Type)

			curDir, err := f.CurrentDir()
			require.NoError(t, err)
			assert.EqualValues(t, "/", curDir)

			size, err := f.FileSize("/server_test.go")
			require.NoError(t, err)
			assert.EqualValues(t, 4, size)

			r, err := f.RetrFrom("/server_test.go", 2)
			require.NoError(t, err)

			buf, err := ioutil.ReadAll(r)
			r.Close()
			require.NoError(t, err)
			assert.EqualValues(t, "st", string(buf))

			err = f.Rename("/server_test.go", "/test.go")
			require.NoError(t, err)

			err = f.MakeDir("/src")
			require.NoError(t, err)

			err = f.Delete("/test.go")
			require.NoError(t, err)

			err = f.ChangeDir("/src")
			require.NoError(t, err)

			curDir, err = f.CurrentDir()
			require.NoError(t, err)
			assert.EqualValues(t, "/src", curDir)

			require.NoError(t, f.Stor("server_test.go", strings.NewReader(content)))

			r, err = f.Retr("/src/server_test.go")
			require.NoError(t, err)

			buf, err = ioutil.ReadAll(r)
			r.Close()
			require.NoError(t, err)
			assert.EqualValues(t, "test", string(buf))

			err = f.RemoveDir("/src")
			require.NoError(t, err)

			curDir, err = f.CurrentDir()
			require.NoError(t, err)
			assert.EqualValues(t, "/", curDir)

			require.NoError(t, f.Stor(" file_name .test", strings.NewReader("tttt")))
			require.NoError(t, f.Delete(" file_name .test"))

			err = f.Quit()
			require.NoError(t, err)

			break
		}
	})
}

func TestLogin(t *testing.T) {
	err := os.MkdirAll("./testdata", os.ModePerm)
	require.NoError(t, err)

	perm := ftp.NewSimplePerm("test", "test")
	driver, err := file.NewDriver("./testdata")
	require.NoError(t, err)

	// Server options without hostname or port
	opt := &ftp.Options{
		Name:   "test ftpd",
		Driver: driver,
		Auth: &ftp.SimpleAuth{
			Name:     "admin",
			Password: "admin",
		},
		Perm:   perm,
		Logger: new(ftp.DiscardLogger),
	}

	// Start the listener
	l, err := net.Listen("tcp", ":2123")
	require.NoError(t, err)

	// Start the server using the listener
	s, err := ftp.NewServer(opt)
	require.NoError(t, err)
	go func() {
		err := s.Serve(l)
		assert.EqualError(t, err, ftp.ErrServerClosed.Error())
	}()

	// Give server 0.5 seconds to get to the listening state
	timeout := time.NewTimer(time.Millisecond * 500)
	for {
		f, err := ftpCli.Connect("localhost:2123")
		if err != nil && len(timeout.C) == 0 { // Retry errors
			continue
		}

		require.NoError(t, err)
		require.NoError(t, f.Login("admin", "admin"))
		require.Error(t, f.Login("admin", ""))
		require.NoError(t, f.Quit())

		break
	}

	require.NoError(t, s.Shutdown())
}
