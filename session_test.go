// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestConnBuildPath(t *testing.T) {
	c := &Session{
		curDir: "",
	}
	pathtests := []struct {
		in  string
		out string
	}{
		{"/", "/"},
		{"one.txt", "/one.txt"},
		{"/files/two.txt", "/files/two.txt"},
		{"files/two.txt", "/files/two.txt"},
		{"/../../../../etc/passwd", "/etc/passwd"},
		{"rclone-test-roxarey8facabob5tuwetet4/hello? sausage/êé/Hello, 世界/ \" ' @ < > & ? + ≠/z.txt", "/rclone-test-roxarey8facabob5tuwetet4/hello? sausage/êé/Hello, 世界/ \" ' @ < > & ? + ≠/z.txt"},
	}
	for _, tt := range pathtests {
		t.Run(tt.in, func(t *testing.T) {
			s := c.buildPath(tt.in)
			if s != tt.out {
				t.Errorf("got %q, want %q", s, tt.out)
			}
		})
	}
}

type mockConn struct {
	ip   net.IP
	port int
}

func (m mockConn) Read(_ []byte) (int, error) {
	return 0, nil
}

func (m mockConn) Write(_ []byte) (int, error) {
	return 0, nil
}

func (m mockConn) Close() error {
	return nil
}

func (m mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{
		IP:   m.ip,
		Port: m.port,
	}
}

func (m mockConn) RemoteAddr() net.Addr {
	return nil
}

func (m mockConn) SetDeadline(_ time.Time) error {
	return nil
}

func (m mockConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (m mockConn) SetWriteDeadline(_ time.Time) error {
	return nil
}

type mockDataSocket struct{ closed bool }

func (m *mockDataSocket) Host() string                        { return "" }
func (m *mockDataSocket) Port() int                           { return 0 }
func (m *mockDataSocket) Read(_ []byte) (int, error)          { return 0, nil }
func (m *mockDataSocket) ReadFrom(_ io.Reader) (int64, error) { return 0, nil }
func (m *mockDataSocket) Write(p []byte) (int, error)         { return len(p), nil }
func (m *mockDataSocket) Close() error                        { m.closed = true; return nil }

func TestSessionCloseClosesDataConn(t *testing.T) {
	dataConn := &mockDataSocket{}
	sess := &Session{Conn: mockConn{}, dataConn: dataConn}

	if err := sess.Close(); err != nil {
		t.Fatalf("close session: %v", err)
	}
	if !dataConn.closed {
		t.Fatal("expected data connection to be closed")
	}
	if sess.dataConn != nil {
		t.Fatal("expected data connection to be cleared")
	}
}

func TestPassiveListenIP(t *testing.T) {
	c := &Session{
		server: &Server{
			Options: &Options{
				PublicIP: "1.1.1.1",
			},
		},
	}
	if c.passiveListenIP() != "1.1.1.1" {
		t.Fatalf("Expected passive listen IP to be 1.1.1.1 but got %s", c.passiveListenIP())
	}

	c = &Session{
		Conn: mockConn{
			ip: net.IPv4(1, 1, 1, 1),
		},
		server: &Server{
			Options: &Options{},
		},
	}
	if c.passiveListenIP() != "1.1.1.1" {
		t.Fatalf("Expected passive listen IP to be 1.1.1.1 but got %s", c.passiveListenIP())
	}
}
