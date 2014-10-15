// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package circular

// A Buffer is a fixed size buffer with Read and Write methods. When more than
// the allotted amount of data is written, the buffer will only retain the last
// size bytes of unread data. The zero value for Buffer is a buffer of size 0;
// all writes are discarded and there is never any data to read. This implementation
// is not thread-safe.
type Buffer struct {
	size   int    // desired maxinum size of buffer
	start  int    // the current read position
	end    int    // the current write position
	buffer []byte // the actual buffer
}

// TODO: this implementation is very simple and can be improved upon for
// performance reasons. https://en.wikipedia.org/wiki/Circular_buffer

// NewBuffer returns a pointer to a Buffer with a maximum size of size.
func NewBuffer(size int) *Buffer {
	return &Buffer{
		size:   size + 1,
		buffer: make([]byte, size+1),
	}
}

// Write writes p[]byte to the buffer and returns the number of bytes written.
// Only the last b.size bytes are kept.
func (b *Buffer) Write(p []byte) (n int, err error) {

	for _, bbyte := range p {
		b.writebyte(bbyte)
	}
	return len(p), nil
}

// Read reads up to len(p) bytes in to p; the actual number of bytes read is
// returned (n) along with any error condition.
func (b *Buffer) Read(p []byte) (n int, err error) {

	for i := 0; i < len(p); i++ {
		if b.IsEmpty() {
			break
		}
		p[i] = b.readbyte()
		n++
	}
	return n, nil
}

// writebyte writes bbyte to the buffer
func (b *Buffer) writebyte(bbyte byte) {
	b.buffer[b.end] = bbyte
	b.end = (b.end + 1) % b.size
	if b.end == b.start {
		b.start = (b.start + 1) % b.size
	}
}

// readbyte reads a byte from the buffer. Callers must ensure that the buffer
// is not IsEmpty()
func (b *Buffer) readbyte() (bbyte byte) {
	bbyte = b.buffer[b.start]
	b.start = (b.start + 1) % b.size
	return bbyte
}

// IsEmpty will return true if the buffer is empty
func (b *Buffer) IsEmpty() bool {
	return b.end == b.start
}

// IsFull will return true if the buffer is full
func (b *Buffer) IsFull() bool {
	return (b.end+1)%b.size == b.start
}
