package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ArchiveReader provides a convenient interface to files, directories and tgz
// archives, both local and remote. In all cases, it allows sequential access
// to the file(s) specified. It mimics the tar.Reader interface; the Next method
// advances to the next file in the archive, returning the path to the file.
// The ArchiveReader can then be used as an io.Reader to access the file's
// data.
type ArchiveReader interface {
	Next() (string, error)
	io.Reader
}

// NewArchiveReader turns a local path or URL into an instance of the
// appropriate implementation of ArchiveReader, based on whether the path
// specified is a remote URL or local path, gzipped or not, file or directory
// or tarball. If it's a tarball, you can add ?subpath=some/path to restrict
// the archive to a subpath of the tarball.
func NewArchiveReader(path string) (ArchiveReader, error) {
	url, err := url.Parse(path)
	if err != nil {
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
			return NewDirectoryReader(url.Path, fi), nil
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
				return NewSingleFileReader(f.Name(), backup), nil
			}
			// Make a new gzip reader with same data that doesn't tee
			fz, err := gzip.NewReader(backup)
			return NewTarballReader(fz, url.Query().Get("subpath")), nil
		default:
			return nil, nil
		}
	} else {
		// Remote path; always a single file
		resp, err := http.Get(url.String())
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, errors.New(resp.Status)
		}
		split := strings.Split(url.Path, "/")
		name := split[len(split)-1]
		tmpBuffer := &bytes.Buffer{}
		safereader := io.TeeReader(resp.Body, tmpBuffer)
		backup := io.MultiReader(tmpBuffer, resp.Body)
		// First, the test
		_, err = gzip.NewReader(safereader)
		if err != nil {
			// Not gzipped, just read it as a single file
			return NewSingleFileReader(name, backup), nil
		}
		// Make a new gzip reader with same data that doesn't tee
		fz, err := gzip.NewReader(backup)
		return NewTarballReader(fz, url.Query().Get("subpath")), nil
	}
}

type fileReader struct {
	r io.Reader
}

type nameHaver struct {
	name string
}

func (n *nameHaver) Name() string {
	return n.name
}

// SingleFileReader wraps a single file in an ArchiveReader, i.e., Next() may
// only be called once before io.EOF is returned.
type SingleFileReader struct {
	path string
	name string
	read bool
	*fileReader
}

func NewSingleFileReader(path string, reader io.Reader) *SingleFileReader {
	root := filepath.Dir(path)
	name := filepath.Base(path)
	return &SingleFileReader{root, name, false, &fileReader{reader}}
}

// TarballReader wraps a tar archive in an ArchiveReader, presenting an
// interface nearly identical to tar.Reader.
type TarballReader struct {
	tarreader *tar.Reader
	subpath   string
}

func NewTarballReader(reader io.Reader, subpath string) *TarballReader {
	return &TarballReader{tar.NewReader(reader), subpath}
}

// DirectoryReader wraps a local directory in an ArchiveReader.
type DirectoryReader struct {
	root    string
	curpath string
	entries []os.FileInfo
	dirs    map[string]os.FileInfo
	*fileReader
	*nameHaver
}

func NewDirectoryReader(path string, fi os.FileInfo) *DirectoryReader {
	return &DirectoryReader{
		root:       path,
		curpath:    filepath.Dir(path),
		entries:    []os.FileInfo{fi},
		dirs:       map[string]os.FileInfo{},
		fileReader: &fileReader{nil},
		nameHaver:  &nameHaver{""},
	}
}

// The first time, Next provides access to the underlying file. The second
// time, it returns io.EOF.
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
		if len(r.dirs) == 0 {
			return "", io.EOF
		}
		for k, _ := range r.dirs {
			entries, err := ioutil.ReadDir(k)
			if err != nil {
				return "", err
			}
			r.entries = entries
			r.curpath = k
			// We just want the first directory
			delete(r.dirs, k)
			break
		}
	}
	// Pop off the first item
	now := r.entries[0]
	r.entries = r.entries[1:]
	nowname := filepath.Join(r.curpath, now.Name())
	// If it's a directory, append to the queue
	if now.IsDir() {
		r.dirs[nowname] = now
		return r.Next()
	} else {
		// It's a file!
		f, err := os.Open(nowname)
		if err != nil {
			return "", err
		}
		r.r = f
		return strings.TrimPrefix(nowname, r.root+"/"), nil
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
	if !strings.HasPrefix(hdr.Name, r.subpath) {
		return r.Next()
	}
	return strings.TrimPrefix(hdr.Name, r.subpath+"/"), nil
}

func (r *TarballReader) Read(b []byte) (int, error) {
	return r.tarreader.Read(b)
}

// ArchiveIterator gives a slightly cleaner interface for handling
// ArchiveReaders, similar to that of bufio.Scanner. One may iterate over an
// ArchiveIterator using the following pattern:
//
//	for iterator.Iterate(filterfunc) {
//      ...
//	}
//
// Within the loop, iterate.Name and iterate.Read provide access to the
// current file. If filterfunc is specified, it will be called with each
// file's name, and only those resulting in a value of true will be returned.
type ArchiveIterator struct {
	Reader *ArchiveReader
	name   string
	err    error
}

// NewArchiveIterator creates an ArchiveReader and wraps it in an
// ArchiveIterator.
func NewArchiveIterator(path string) (*ArchiveIterator, error) {
	reader, err := NewArchiveReader(path)
	if err != nil {
		return nil, err
	}
	return &ArchiveIterator{
		Reader: &reader,
	}, nil
}

// Name returns the name of the current file.
func (it *ArchiveIterator) Name() string {
	return it.name
}

// Iterate advances the underlying ArchiveReader and returns a boolean
// indicating whether another file is available. If filterfunc is specified,
// every file returned will have a name that satisfies it.
func (it *ArchiveIterator) Iterate(filter func(string) bool) bool {
	reader := *it.Reader
	for {
		name, err := reader.Next()
		if err != nil {
			it.err = err
			return false
		}
		if filter == nil || filter(name) {
			it.name = name
			return true
		}
	}
}

func (it *ArchiveIterator) Read(b []byte) (int, error) {
	return (*it.Reader).Read(b)
}

// Err returns the error, if any, that arose during iteration. If the error
// was an ordinary io.EOF, indicating that file reading completed normally,
// Err will return nil.
func (it *ArchiveIterator) Err() error {
	if it.err != nil && it.err != io.EOF {
		return it.err
	}
	return nil
}
