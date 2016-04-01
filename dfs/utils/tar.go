// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"io"

	"gopkg.in/pipe.v2"
)

type TarStreamMerger struct {
	p *pipe.Line
}

func NewTarStreamMerger(w io.Writer) *TarStreamMerger {
}

/*
const (
	// Tar archives comprise 512-byte blocks
	blocksize = 512
	// The terminator of a tar archive is two 512-byte blocks of zeroes, so that's our buffer
	bufsize = blocksize << 1
)

var (
	zeroes = make([]byte, bufsize)
	// ErrClosed is returned when a write happens after the stream is closed
	ErrClosed = errors.New("TarStreamMerger: write after closed")
)

// TarStreamMerger is a utility to combine multiple tar streams into a single
// tarball without using much intermediate memory or disk. It accomplishes this
// by omitting the end-of-archive terminator of each stream, then simply
// concatenating them to the provided writer.
type TarStreamMerger struct {
	out    io.Writer
	buffer [bufsize]byte
	closed bool
}

// NewTarStreamMerger creates a new TarStreamMerger
func NewTarStreamMerger(w io.Writer) *TarStreamMerger {
	return &TarStreamMerger{out: w}
}

// Append appends a new tar stream to the merger
func (m *TarStreamMerger) Append(tarStream io.Reader) error {
	if m.closed {
		return ErrClosed
	}

	reader := bufio.NewReader(tarStream)

	// Clear out the buffer
	buffer := m.buffer[:]
	copy(buffer, zeroes)

	for {
		if _, err := io.ReadFull(tarStream, buffer); err != nil {
			// we got io.EOF, the only thing io.ReadFull can return
			if bytes.Equal(buffer, zeroes) {
				// We got a terminator; don't write it
				return nil
			}
			if bytes.Equal(buffer[blocksize:], zeroes[0:blocksize]) {
				// Second block is a zero; check for a terminator
				if b, err := reader.Peek(blocksize); err != nil || bytes.Equal(b, zeroes[0:blocksize]) {
					m.out.Write(buffer[0:blocksize])
					return nil
				}
			}
			if _, err := m.out.Write(buffer); err != nil {
				return err
			}
			// Reset the buffer again
			copy(buffer, zeroes)
		}
	}

}

// Close finalizes the merged tar stream by appending the archive terminator.
func (m *TarStreamMerger) Close() error {
	if m.closed {
		return nil
	}
	m.closed = true
	// Write a terminator
	_, err := m.out.Write(zeroes)
	return err
}
*/
