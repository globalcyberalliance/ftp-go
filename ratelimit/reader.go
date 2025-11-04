// Copyright 2020 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ratelimit

import (
	"fmt"
	"io"
)

type reader struct {
	reader  io.Reader
	limiter *Limiter
}

// Read reads data from the underlying reader while respecting the rate limit defined in the limiter.
// It returns the number of bytes read and an error if the read operation fails.
func (r *reader) Read(buf []byte) (int, error) {
	r.limiter.Wait(len(buf))

	n, err := r.reader.Read(buf)
	if err != nil {
		return n, fmt.Errorf("read: %w", err)
	}

	return n, nil
}

// Reader returns a reader with limiter.
func Reader(r io.Reader, l *Limiter) io.Reader {
	return &reader{
		reader:  r,
		limiter: l,
	}
}
