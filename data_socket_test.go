package ftp

import (
	"errors"
	"net"
	"strconv"
	"testing"
)

type remoteConn struct {
	mockConn

	remote net.Addr
}

func (c remoteConn) RemoteAddr() net.Addr {
	return c.remote
}

func TestNewActiveSocketOnlyDialsTheClient(t *testing.T) {
	listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		t.Fatalf("listen: %v", listenErr)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	tests := []struct {
		name    string
		client  net.Addr
		host    string
		wantErr error
	}{
		{"client's own address", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5000}, "127.0.0.1", nil},
		{"another host", &net.TCPAddr{IP: net.ParseIP("203.0.113.7"), Port: 5000}, "127.0.0.1", ErrForeignDataAddress},
		{"hostname", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5000}, "localhost", ErrForeignDataAddress},
		{"no client address", nil, "127.0.0.1", ErrForeignDataAddress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess := &Session{
				Conn:   remoteConn{remote: tt.client},
				server: &Server{logger: &DiscardLogger{}},
			}

			socket, err := newActiveSocket(sess, tt.host, port)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("newActiveSocket(%s:%s) error = %v, want %v", tt.host, strconv.Itoa(port), err, tt.wantErr)
			}

			if socket != nil {
				_ = socket.Close()
			}
		})
	}
}

func TestClientDialAddrZones(t *testing.T) {
	client := &net.TCPAddr{IP: net.ParseIP("fe80::1"), Port: 5000, Zone: "eth0"}

	tests := []struct {
		host string
		want string
	}{
		{"fe80::1", "fe80::1%eth0"},
		{"fe80::1%eth0", "fe80::1%eth0"},
		{"fe80::1%eth1", ""},
	}

	for _, tt := range tests {
		addr, ok := clientDialAddr(client, tt.host)
		if got := addr.String(); ok != (tt.want != "") || ok && got != tt.want {
			t.Errorf("clientDialAddr(%s) = %s, %v, want %q", tt.host, got, ok, tt.want)
		}
	}
}
