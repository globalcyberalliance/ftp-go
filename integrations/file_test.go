// Copyright 2020 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package integrations

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/globalcyberalliance/ftp-go"
	"github.com/globalcyberalliance/ftp-go/driver/file"
	ftpCLI "github.com/jlaffaye/ftp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDataDir     = "./testdata"
	testFileContent = "test"
	testServerPort  = 2122
	testTimeout     = 500 * time.Millisecond
	testPassword    = "admin"
	testUsername    = "admin"
)

func TestFileDriver(t *testing.T) {
	// Setup test environment.
	err := os.MkdirAll(testDataDir, os.ModePerm)
	require.NoError(t, err)

	// Create a test server.
	perm := ftp.NewSimplePerm("test", "test")
	driver, err := file.NewDriver(testDataDir)
	require.NoError(t, err)

	opt := &ftp.Options{
		Name:   "test ftpd",
		Driver: driver,
		Perm:   perm,
		Port:   testServerPort,
		Auth: &ftp.SimpleAuth{
			Name:     testUsername,
			Password: testPassword,
		},
		Logger: new(ftp.DiscardLogger),
	}

	runServer(t, opt, nil, func() {
		client := connectWithRetry(t)
		defer client.Quit()

		t.Run("Authentication", func(t *testing.T) {
			// Test valid login already happened in connectWithRetry.

			// Test invalid password.
			assert.Error(t, client.Login(testUsername, ""))
		})

		t.Run("FileUploadAndList", func(t *testing.T) {
			const testFileName = "server_test.go"

			// Upload file.
			err := client.Stor(testFileName, strings.NewReader(testFileContent))
			require.NoError(t, err)

			// Verify file was written to disk.
			filePath := filepath.Join(testDataDir, testFileName)
			bs, err := os.ReadFile(filePath)
			require.NoError(t, err)
			assert.Equal(t, testFileContent, string(bs))

			// List files.
			names, err := client.NameList("/")
			require.NoError(t, err)
			assert.Contains(t, names, testFileName)

			// Get detailed file list.
			entries, err := client.List("/")
			require.NoError(t, err)
			assert.Len(t, entries, 1)
			assert.Equal(t, testFileName, entries[0].Name)
			assert.EqualValues(t, 4, entries[0].Size)
			assert.Equal(t, ftpCLI.EntryTypeFile, entries[0].Type)
		})

		t.Run("FileSize", func(t *testing.T) {
			const testFileName = "server_test.go"

			size, err := client.FileSize("/" + testFileName)
			require.NoError(t, err)
			assert.EqualValues(t, 4, size)
		})

		t.Run("PartialFileRetrieval", func(t *testing.T) {
			const (
				testFileName     = "server_test.go"
				partialReadStart = 2
			)

			r, err := client.RetrFrom("/"+testFileName, partialReadStart)
			require.NoError(t, err)
			defer r.Close()

			buf, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.Equal(t, "st", string(buf))
		})

		t.Run("FileRenameAndDelete", func(t *testing.T) {
			const (
				oldName = "/server_test.go"
				newName = "/test.go"
			)

			err := client.Rename(oldName, newName)
			require.NoError(t, err)

			err = client.Delete(newName)
			require.NoError(t, err)
		})

		t.Run("DirectoryOperations", func(t *testing.T) {
			const (
				testDirName  = "/src"
				testFileName = "server_test.go"
			)

			// Verify current directory.
			curDir, err := client.CurrentDir()
			require.NoError(t, err)
			assert.Equal(t, "/", curDir)

			// Create directory.
			err = client.MakeDir(testDirName)
			require.NoError(t, err)

			// Change to new directory.
			err = client.ChangeDir(testDirName)
			require.NoError(t, err)

			curDir, err = client.CurrentDir()
			require.NoError(t, err)
			assert.Equal(t, testDirName, curDir)

			// Upload file to directory.
			err = client.Stor(testFileName, strings.NewReader(testFileContent))
			require.NoError(t, err)

			// Retrieve file from directory.
			r, err := client.Retr(testDirName + "/" + testFileName)
			require.NoError(t, err)
			defer r.Close()

			buf, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.Equal(t, testFileContent, string(buf))

			// Remove directory.
			err = client.RemoveDir(testDirName)
			require.NoError(t, err)

			// Verify we're back at root.
			curDir, err = client.CurrentDir()
			require.NoError(t, err)
			assert.Equal(t, "/", curDir)
		})

		t.Run("FileNamesWithSpaces", func(t *testing.T) {
			const (
				fileNameWithSpaces = " file_name .test"
				fileContent        = "tttt"
			)

			err := client.Stor(fileNameWithSpaces, strings.NewReader(fileContent))
			require.NoError(t, err)

			err = client.Delete(fileNameWithSpaces)
			require.NoError(t, err)
		})
	})
}

// connectWithRetry establishes a connection to the test FTP server with retry logic.
func connectWithRetry(t *testing.T) *ftpCLI.ServerConn {
	t.Helper()

	var client *ftpCLI.ServerConn
	var err error

	serverAddr := fmt.Sprintf("localhost:%d", testServerPort)
	timeout := time.After(testTimeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			require.NoError(t, err, "Failed to connect to server within timeout")
			return nil
		case <-ticker.C:
			client, err = ftpCLI.Dial(serverAddr)
			if err == nil {
				require.NoError(t, client.Login(testUsername, testPassword))
				return client
			}
		}
	}
}

func TestLogin(t *testing.T) {
	err := os.MkdirAll("./testdata", os.ModePerm)
	require.NoError(t, err)

	perm := ftp.NewSimplePerm("test", "test")
	driver, err := file.NewDriver("./testdata")
	require.NoError(t, err)

	// Server options without a hostname or port.
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

	// Start the listener.
	l, err := net.Listen("tcp", ":2123")
	require.NoError(t, err)

	// Start the server using the listener.
	s, err := ftp.NewServer(opt)
	require.NoError(t, err)
	go func() {
		serveErr := s.Serve(l)
		assert.EqualError(t, serveErr, ftp.ErrServerClosed.Error())
	}()

	// Give the server 0.5 seconds to reach the listening state.
	timeout := time.NewTimer(time.Millisecond * 500)
	for {
		f, dialErr := ftpCLI.Dial("localhost:2123")
		if dialErr != nil && len(timeout.C) == 0 { // Retry errors
			continue
		}

		require.NoError(t, dialErr)
		require.NoError(t, f.Login("admin", "admin"))
		require.Error(t, f.Login("admin", ""))
		require.NoError(t, f.Quit())

		break
	}

	require.NoError(t, s.Shutdown())
}
