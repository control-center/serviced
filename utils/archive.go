package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
)

type ArchiveReader interface {
	Next() (string, error)
	io.Reader
}

func NewArchiveReader(path string) (ArchiveReader, error) {
	url, err := url.Parse(path)
	if err != nil {
		fmt.Println("err is nil")
		return nil, err
	}
	if url.Scheme == "" && url.Host == "" {
		// Local path
		f, err := os.Open(url.Path)
		if err != nil {
			return nil, err
		}
		fi, err := f.Stat()
		if err != nil {
			return nil, err
		}
		switch mode := fi.Mode(); {
		case mode.IsDir():
			// Directory
			defer f.Close()
			return &DirectoryReader{filepath.Dir(path), []os.FileInfo{fi},
				&fileReader{nil}}, nil
		case mode.IsRegular():
			// File
			// Assume gzipped, error will tell us otherwise
			// gzip.NewReader consumes the Reader, so we'll tee off the data
			// until we know. Not forever, though, because in good scenario
			// would eat memory.
			tmpBuffer := &bytes.Buffer{}
			safereader := io.TeeReader(f, tmpBuffer)
			backup := io.MultiReader(tmpBuffer, f)
			// First, the test
			_, err := gzip.NewReader(safereader)
			if err != nil {
				// Not gzipped, just read it as a single file
				return &SingleFileReader{f.Name(), false, &fileReader{backup}}, nil
			}
			// Make a new gzip reader with same data that doesn't tee
			fz, err := gzip.NewReader(backup)
			return &TarballReader{tar.NewReader(fz)}, nil
		}
	} else {
		// Remote path; always a single file
		fmt.Println("REMOTE PATH")
	}
	return nil, nil
}

type fileReader struct {
	r io.Reader
}

type SingleFileReader struct {
	name string
	read bool
	*fileReader
}

type TarballReader struct {
	tarreader *tar.Reader
}

type DirectoryReader struct {
	path    string
	entries []os.FileInfo
	*fileReader
}

func (r *SingleFileReader) Next() (string, error) {
	if r.read == false {
		r.read = true
		return r.name, nil
	}
	return "", io.EOF
}

func (r *SingleFileReader) Read(b []byte) (int, error) {
	return r.r.Read(b)
}

func (r *DirectoryReader) Next() (string, error) {
	if len(r.entries) == 0 {
		return "", io.EOF
	}
	// Pop off the first item
	now := r.entries[0]
	r.entries = r.entries[1:]
	// If it's a directory, append to the queue
	nowname := filepath.Join(r.path, now.Name())
	if now.IsDir() {
		r.path = nowname
		entries, err := ioutil.ReadDir(nowname)
		if err != nil {
			return "", err
		}
		r.entries = append(entries, r.entries...)
		return r.Next()
	} else {
		// It's a file!
		f, err := os.Open(nowname)
		if err != nil {
			return "", err
		}
		r.r = f
		return nowname, nil
	}
}

func (r *DirectoryReader) Read(b []byte) (int, error) {
	return r.r.Read(b)
}

func (r *TarballReader) Next() (string, error) {
	hdr, err := r.tarreader.Next()
	if err != nil {
		return "", err
	}
	if hdr.FileInfo().IsDir() {
		return r.Next()
	}
	return hdr.Name, nil
}

func (r *TarballReader) Read(b []byte) (int, error) {
	return r.tarreader.Read(b)
}
