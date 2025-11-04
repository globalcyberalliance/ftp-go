// Copyright 2020 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ratelimit

import (
	"fmt"
	"io"
)

type writer struct {
	writer  io.Writer
	limiter *Limiter
}

// Write writes the provided byte slice to the underlying writer while respecting the rate limit defined in the limiter.
// It returns the number of bytes written and an error if the write operation fails.
func (w *writer) Write(buf []byte) (int, error) {
	w.limiter.Wait(len(buf))

	n, err := w.writer.Write(buf)
	if err != nil {
		return 0, fmt.Errorf("write: %w", err)
	}

	return n, nil
}

// Writer returns a writer with limiter.
func Writer(w io.Writer, l *Limiter) io.Writer {
	return &writer{
		writer:  w,
		limiter: l,
	}
}
