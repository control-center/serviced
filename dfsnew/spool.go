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

package dfs

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"sync"
)

// Spooler is the interface for the Spool
type Spooler interface {
	Write(p []byte) (n int, err error)
	WriteTo(w io.Writer) (n int64, err error)
	Reset() error
	Size() int64
	Close() error
}

// Spool calculates the size of a writer stream and tees the output to
// another stream.
type Spool struct {
	locker   sync.Locker
	file     *os.File
	gzwriter *gzip.Writer
	size     int64
	capacity int
}

// NewSpool instantiates a new spool
func NewSpool(dir string) (*Spool, error) {
	// create a tempfile
	file, err := ioutil.TempFile(dir, "spool-")
	if err != nil {
		return nil, err
	}
	// gzip all incoming writes
	gzwriter := gzip.NewWriter(file)
	return &Spool{&sync.Mutex{}, file, gzwriter, 0, 1024 * 1024}, nil
}

// Write compresses byte data and writes to disk, while keeping track of the
// total number of bytes written
func (s *Spool) Write(p []byte) (n int, err error) {
	s.locker.Lock()
	defer s.locker.Unlock()
	n, err = s.gzwriter.Write(p)
	s.size += int64(n)
	return
}

// WriteTo writes the filedata back into another writer and resets the file.
func (s *Spool) WriteTo(w io.Writer) (n int64, err error) {
	s.locker.Lock()
	defer s.locker.Unlock()
	defer s.reset()
	if err := s.gzwriter.Close(); err != nil {
		return 0, err
	}
	s.gzwriter.Close()
	if _, err := s.file.Seek(0, 0); err != nil {
		return 0, err
	}
	gzreader, err := gzip.NewReader(s.file)
	if err != nil {
		return 0, err
	}
	defer gzreader.Close()
	for {
		buf := make([]byte, s.capacity)
		rn, rerr := gzreader.Read(buf)
		if rerr == io.EOF || rerr == nil {
			wn, werr := w.Write(buf[:rn])
			n += int64(wn)
			if werr != nil {
				return n, werr
			} else if rerr != nil {
				return n, nil
			}
		} else {
			return n, rerr
		}
	}
}

// Reset truncates the file an resets the file
func (s *Spool) Reset() error {
	s.locker.Lock()
	defer s.locker.Unlock()
	return s.reset()
}

func (s *Spool) reset() error {
	s.gzwriter.Close()
	if _, err := s.file.Seek(0, 0); err != nil {
		return err
	}
	if err := s.file.Truncate(0); err != nil {
		return err
	}
	s.gzwriter.Reset(s.file)
	s.size = 0
	return nil
}

// Size is the current size of the file
func (s *Spool) Size() int64 {
	s.locker.Lock()
	defer s.locker.Unlock()
	return s.size
}

// Close closes the file handle and removes the file.
func (s *Spool) Close() error {
	s.locker.Lock()
	defer s.locker.Unlock()
	defer os.Remove(s.file.Name())
	s.reset()
	return s.file.Close()
}
