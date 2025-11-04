// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/globalcyberalliance/ftp-go/ratelimit"
)

const version = "2.0beta"

// ErrServerClosed is returned by ListenAndServe() or Serve() when a shutdown was requested.
var ErrServerClosed = errors.New("ftp: Server closed")

type (
	// Options contains parameters for server.NewServer().
	Options struct {
		// The driver that will be used to handle files persistent
		Driver Driver

		// How to handle authentication requests
		Auth Auth

		// How to handle the perm controls
		Perm Perm

		// A logger implementation, if nil the StdLogger is used.
		Logger Logger

		// The server's supported commands. Defaults to defaultCommands.
		Commands map[string]Command

		// Server Name, defaults to Go Ftp Server.
		Name string

		// The hostname that the FTP server should listen on. Optional, defaults to
		// "::", which means all hostnames on ipv4 and ipv6.
		Hostname string

		// Public IP of the server,
		PublicIP string

		// Disable the use of passive ports.
		DisablePassive bool

		// Passive ports.
		PassivePorts string

		// WelcomeMessage is a customizable message displayed to users upon successful connection to the server.
		WelcomeMessage string

		// The port that the FTP should listen on. Optional, defaults to 3000. In
		// a production environment you will probably want to change this to 21.
		Port int

		// Rate Limit per connection bytes per second, 0 means no limit.
		RateLimit int64

		// Timeout is used to restrict the total length of a session.
		Timeout time.Duration

		// CommandsMu controls access to the Commands map.
		CommandsMu sync.RWMutex

		// TLSConfig if supplied, will enable TLS on the server.
		TLSConfig *tls.Config

		// If true, TLS is used in RFC4217 mode.
		ExplicitFTPS bool

		// If true, client must upgrade to TLS before sending any other command.
		ForceTLS bool
	}

	// Server is the root of your FTP application. You should instantiate one
	// of these and call ListenAndServe() to start accepting client connections.
	//
	// Always use the NewServer() method to create a new Server.
	Server struct {
		*Options

		logger       Logger
		listener     net.Listener
		ctx          context.Context
		cancel       context.CancelFunc
		rateLimiter  *ratelimit.Limiter                                       // Rate limiter per connection.
		ConnCallback func(ctx context.Context, conn net.Conn) net.Conn        // Optional callback for wrapping net.Conn before handling.
		ConnContext  func(ctx context.Context, conn net.Conn) context.Context // Optional callback for wrapping context.Context before handling.
		listenTo     string
		feats        string
		notifiers    notifierList
	}

	// serverConn is used to wrap a handle with context.
	serverConn struct {
		net.Conn

		ctx    context.Context
		cancel context.CancelFunc
	}
)

// optsWithDefaults copies an Options struct into a new struct,
// then adds any default values that are missing and returns the new data.
func optsWithDefaults(opts *Options) *Options {
	var newOpts Options
	if opts == nil {
		opts = &Options{}
	}

	newOpts.Hostname = opts.Hostname
	if opts.Hostname == "" {
		newOpts.Hostname = "::"
	}

	newOpts.Port = opts.Port
	if opts.Port == 0 {
		newOpts.Port = 2121
	}

	newOpts.Name = opts.Name
	if opts.Name == "" {
		newOpts.Name = "Go FTP Server"
	}

	newOpts.WelcomeMessage = opts.WelcomeMessage
	if opts.WelcomeMessage == "" {
		newOpts.WelcomeMessage = defaultWelcomeMessage
	}

	if opts.Auth != nil {
		newOpts.Auth = opts.Auth
	}

	newOpts.Logger = &StdLogger{}
	if opts.Logger != nil {
		newOpts.Logger = opts.Logger
	}

	newOpts.Commands = opts.Commands
	if len(opts.Commands) == 0 {
		newOpts.Commands = DefaultCommands()
	}

	if opts.DisablePassive {
		delete(newOpts.Commands, "PASV")
	}

	newOpts.Timeout = opts.Timeout
	if opts.Timeout.Seconds() <= 0 {
		newOpts.Timeout = defaultTimeout
	}

	newOpts.DisablePassive = opts.DisablePassive
	newOpts.Driver = opts.Driver
	newOpts.ExplicitFTPS = opts.ExplicitFTPS
	newOpts.Perm = opts.Perm
	newOpts.TLSConfig = opts.TLSConfig
	newOpts.PassivePorts = opts.PassivePorts
	newOpts.PublicIP = opts.PublicIP
	newOpts.RateLimit = opts.RateLimit

	return &newOpts
}

// NewServer initialises a new FTP server. Configuration options are provided
// via an instance of Options. Calling this function in your code will
// probably look something like this:
//
//	driver := &MyDriver{}
//	opts   := &server.Options{
//	  Driver: driver,
//	  Auth: auth,
//	  Port: 2000,
//	  Perm: perm,
//	  Hostname: "127.0.0.1",
//	}
//	server, err  := server.NewServer(opts)
func NewServer(opts *Options) (*Server, error) {
	opts = optsWithDefaults(opts)
	if opts.Perm == nil {
		return nil, errors.New("no perm implementation")
	}

	s := &Server{
		Options:  opts,
		listenTo: net.JoinHostPort(opts.Hostname, strconv.Itoa(opts.Port)),
		logger:   opts.Logger,
	}

	feats := "Extensions supported:\n%s"
	featCmds := " UTF8\n"

	for k, v := range s.Commands {
		if v.IsExtend() {
			featCmds = featCmds + " " + k + "\n"
		}
	}

	if opts.TLSConfig != nil {
		featCmds += " AUTH TLS\n PBSZ\n PROT\n"
	}

	s.feats = fmt.Sprintf(feats, featCmds)
	s.rateLimiter = ratelimit.New(opts.RateLimit)

	return s, nil
}

// RegisterNotifier registers a notifier.
func (server *Server) RegisterNotifier(notifier Notifier) {
	server.notifiers = append(server.notifiers, notifier)
}

// NewConn constructs a new object that will handle the FTP protocol over an active net.TCPConn. The TCP connection
// should already be open before it is handed to this function.
func (server *Server) newSession(ctx context.Context, id string, tcpConn net.Conn) *Session {
	return &Session{
		ctx:           ctx,
		id:            id,
		server:        server,
		controlReader: bufio.NewReader(tcpConn),
		controlWriter: bufio.NewWriter(tcpConn),
		curDir:        "/",
		reqUser:       "",
		user:          "",
		renameFrom:    "",
		lastFilePos:   -1,
		closed:        false,
		tls:           false,
		Conn:          tcpConn,
		Data:          make(map[string]any),
	}
}

// ListenAndServe asks a new Server to begin accepting client connections. It accepts no arguments - all configuration
// is provided via the NewServer function.
//
// If the server fails to start for any reason, an error will be returned. Common errors are trying to bind to a
// privileged port or something else is already listening on the same port.
func (server *Server) ListenAndServe() error {
	const protoTCP = "tcp"
	var listener net.Listener
	var err error

	ctx := context.Background()
	conf := &net.ListenConfig{}

	if server.Options.TLSConfig != nil {
		if server.Options.ExplicitFTPS {
			listener, err = conf.Listen(ctx, protoTCP, server.listenTo)
		} else {
			listener, err = tls.Listen(protoTCP, server.listenTo, server.Options.TLSConfig)
		}
	} else {
		listener, err = conf.Listen(ctx, protoTCP, server.listenTo)
	}
	if err != nil {
		return fmt.Errorf("open listener: %w", err)
	}

	server.logger.Printf("", "%s listening on %d", server.Name, server.Port)

	return server.Serve(listener)
}

// Serve accepts connections on a given net.Listener and handles each
// request in a new goroutine.
func (server *Server) Serve(l net.Listener) error {
	server.listener = l
	server.ctx, server.cancel = context.WithCancel(context.Background())
	defer server.cancel()

	sessionID, err := newSessionID()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}

	for {
		rawConn, lErr := server.listener.Accept()
		if lErr != nil {
			return fmt.Errorf("accept connection: %w", lErr)
		}

		var ctx context.Context
		var cancel context.CancelFunc

		if server.Timeout > 0 {
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(server.Timeout))
		} else {
			ctx, cancel = context.WithCancel(context.Background())
		}

		if server.ConnCallback != nil {
			rawConn = server.ConnCallback(ctx, rawConn)
			if rawConn == nil {
				panic("ConnCallback returned nil")
			}
		}

		if server.ConnContext != nil {
			ctx = server.ConnContext(ctx, rawConn)
			if ctx == nil {
				panic("ConnContext returned nil")
			}
		}

		conn := serverConn{
			Conn:   rawConn,
			cancel: cancel,
			ctx:    ctx,
		}

		ftpConn := server.newSession(ctx, sessionID, conn)
		go ftpConn.Serve()
	}
}

// Shutdown will gracefully stop a server. Already connected clients will retain their connections.
func (server *Server) Shutdown() error {
	if server.cancel != nil {
		server.cancel()
	}

	if server.listener != nil {
		if err := server.listener.Close(); err != nil {
			return fmt.Errorf("close listener: %w", err)
		}
	}

	// Server wasn't started.
	return nil
}
