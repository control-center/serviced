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
	"archive/tar"
	"fmt"
	"io"
	"path/filepath"

	"gopkg.in/pipe.v2"
)

// PrefixPath rewrites the paths of all the files within a tar stream to be
// beneath a given prefix.
func PrefixPath(prefix string) pipe.Pipe {
	return pipe.TaskFunc(func(s *pipe.State) error {
		reader := tar.NewReader(s.Stdin)
		writer := tar.NewWriter(s.Stdout)
		defer writer.Close()
		for {
			hdr, err := reader.Next()
			if err == io.EOF {
				// End of the archive
				break
			}
			if err != nil {
				return err
			}
			// Add the prefix
			hdr.Name = filepath.Join(prefix, hdr.Name)
			writer.WriteHeader(hdr)
			io.Copy(writer, reader)
		}
		return nil
	})
}

// StripTerminator echoes the tar archive input minus the final 1024-zero-byte
// terminator. This allows archives to be concatenated.
func stripTerminator() pipe.Pipe {
	return pipe.TaskFunc(func(s *pipe.State) error {
		reader := tar.NewReader(s.Stdin)
		writer := tar.NewWriter(s.Stdout)
		// This is the interesting part. We simply flush the tar writer, but
		// don't close it. This omits the terminator.
		defer writer.Flush()
		for {
			hdr, err := reader.Next()
			if err == io.EOF {
				// End of the archive
				break
			}
			if err != nil {
				return err
			}
			writer.WriteHeader(hdr)
			io.Copy(writer, reader)
		}
		return nil
	})
}

type taskFunc func(s *pipe.State) error

func (f taskFunc) Run(s *pipe.State) error { fmt.Println("RUNNING"); return f(s) }
func (f taskFunc) Kill()                   {}

// Cat concatenates the output of several pipes together.
func Cat(pipes ...pipe.Pipe) pipe.Pipe {
	return pipe.Script(pipes...)
	/*
		return func(s *pipe.State) error {
			stdout := s.Stdout
			for _, p := range pipes {

				s.AddTask(taskFunc(pipe.Line(
					p, pipe.Write(stdout),
				)))
			}
			return nil
		}
	*/
}

func WriteSuffix(suffix []byte) pipe.Pipe {
	return pipe.TaskFunc(func(s *pipe.State) error {
		io.Copy(s.Stdout, s.Stdin)
		_, err := s.Stdout.Write(suffix)
		return err
	})
}

// ConcatTarStreams combines several tar streams into a single tarball.
func ConcatTarStreams(pipes ...pipe.Pipe) pipe.Pipe {
	thepipening := []pipe.Pipe{}
	for _, p := range pipes {
		thepipening = append(thepipening, pipe.Line(p, stripTerminator()))
	}
	return pipe.Line(
		Cat(thepipening...),
		WriteSuffix(make([]byte, 1024)),
	)
}
