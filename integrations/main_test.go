// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package integrations

import (
	"testing"

	"github.com/globalcyberalliance/ftp-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runServer(t *testing.T, opt *ftp.Options, notifiers []ftp.Notifier, execute func()) {
	s, err := ftp.NewServer(opt)
	require.NoError(t, err)
	for _, notifier := range notifiers {
		s.RegisterNotifier(notifier)
	}

	go func() {
		err := s.ListenAndServe()
		assert.EqualError(t, err, ftp.ErrServerClosed.Error())
	}()

	execute()

	require.NoError(t, s.Shutdown())
}
