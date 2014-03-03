package utils_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/zenoss/serviced/utils"

	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var _ = Describe("Archive", func() {

	var (
		path    string
		tempdir string
		r       ArchiveReader
		err     error
	)

	BeforeEach(func() {
		tempdir, _ = ioutil.TempDir("", "test")
	})

	AfterEach(func() {
		os.RemoveAll(tempdir)
	})

	JustBeforeEach(func() {
		r, err = NewArchiveReader(path)
	})

	Describe("a local file", func() {
		Context("the file exists", func() {
			BeforeEach(func() {
				path = filepath.Join(tempdir, "testfile")
				f, _ := os.Create(path)
				defer f.Close()
				f.WriteString("test data")
			})
			It("returns a single file without error", func() {
				name, err := r.Next()
				Expect(name).To(Equal(filepath.Join(tempdir, "testfile")))
				data, _ := ioutil.ReadAll(r)
				Expect(string(data)).To(Equal("test data"))
				Expect(err).To(BeNil())
				_, err = r.Next()
				Expect(err).To(Equal(io.EOF))
			})
		})
		Context("the file does not exist", func() {
			BeforeEach(func() {
				path = "/not/a/path"
			})
			It("returns immediately with an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("a local directory", func() {
		Context("the directory exists", func() {
			BeforeEach(func() {
				path = tempdir
				for i := 0; i < 3; i++ {
					f, _ := os.Create(filepath.Join(tempdir,
						fmt.Sprintf("testfile-%d", i)))
					defer f.Close()
					f.WriteString(fmt.Sprintf("test data %d", i))
				}
				subdir := filepath.Join(tempdir, "zzz")
				os.MkdirAll(subdir, 0777)
				f, _ := os.Create(filepath.Join(subdir, "hi"))
				defer f.Close()
			})
			It("returns all files without error", func() {
				for i := 0; i < 3; i++ {
					name, err := r.Next()
					Expect(name).To(Equal(filepath.Join(tempdir,
						fmt.Sprintf("testfile-%d", i))))
					data, _ := ioutil.ReadAll(r)
					Expect(string(data)).To(Equal(
						fmt.Sprintf("test data %d", i)))
					Expect(err).To(BeNil())
				}
				name, err := r.Next()
				Expect(name).To(Equal(filepath.Join(tempdir, "zzz", "hi")))
				_, err = r.Next()
				Expect(err).To(Equal(io.EOF))
			})
		})
		Context("the directory does not exist", func() {
			// Technically, this is identical to the single file case.
			BeforeEach(func() {
				path = "/not/a/path"
			})
			It("returns immediately with an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("a local tar.gz", func() {
		BeforeEach(func() {
			path = tempdir + "-archive.tgz"
			archdir := filepath.Join(tempdir, "archive")
			os.MkdirAll(archdir, 0777)
			for i := 0; i < 3; i++ {
				f, _ := os.Create(filepath.Join(archdir,
					fmt.Sprintf("testfile-%d", i)))
				defer f.Close()
				f.WriteString(fmt.Sprintf("test data %d", i))
			}
			subdir := filepath.Join(archdir, "zzz")
			os.MkdirAll(subdir, 0777)
			f, _ := os.Create(filepath.Join(subdir, "hi"))
			defer f.Close()
			exec.Command("tar", "czf", path, archdir).Run()
		})

		It("returns all files with no error", func() {
			archdir := filepath.Join(tempdir, "archive")
			for i := 0; i < 3; i++ {
				name, err := r.Next()
				Expect(name).To(Equal(filepath.Join(archdir,
					fmt.Sprintf("testfile-%d", i))[1:]))
				data, _ := ioutil.ReadAll(r)
				Expect(string(data)).To(Equal(
					fmt.Sprintf("test data %d", i)))
				Expect(err).To(BeNil())
			}
			name, err := r.Next()
			Expect(name).To(Equal(filepath.Join(archdir, "zzz", "hi")[1:]))
			_, err = r.Next()
			Expect(err).To(Equal(io.EOF))
		})
	})

	Context("remote files", func() {

		var (
			server *httptest.Server
			addr   string
		)

		BeforeEach(func() {
			server = httptest.NewServer(http.FileServer(http.Dir(tempdir)))
			addr = server.Listener.Addr().String()
		})

		AfterEach(func() {
			server.Close()
		})

		Describe("a remote file", func() {
			Context("the file exists", func() {
				BeforeEach(func() {
					fpath := filepath.Join(tempdir, "testfile")
					f, _ := os.Create(fpath)
					defer f.Close()
					f.WriteString("test data")
					path = fmt.Sprintf("http://%s/testfile", addr)
				})
				It("returns a single file without error", func() {
					name, err := r.Next()
					Expect(name).To(Equal("testfile"))
					data, _ := ioutil.ReadAll(r)
					Expect(string(data)).To(Equal("test data"))
					Expect(err).To(BeNil())
					_, err = r.Next()
					Expect(err).To(Equal(io.EOF))
				})
			})
			Context("the file does not exist", func() {
				BeforeEach(func() {
					path = fmt.Sprintf("http://%s/not/a/thing", addr)
				})
				It("returns with an error", func() {
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("a remote tar.gz", func() {
			BeforeEach(func() {
				tarball := filepath.Join(tempdir, "archive.tgz")
				archdir := filepath.Join(tempdir, "archive")
				os.MkdirAll(archdir, 0777)
				for i := 0; i < 3; i++ {
					f, _ := os.Create(filepath.Join(archdir,
						fmt.Sprintf("testfile-%d", i)))
					defer f.Close()
					f.WriteString(fmt.Sprintf("test data %d", i))
				}
				subdir := filepath.Join(archdir, "zzz")
				os.MkdirAll(subdir, 0777)
				f, _ := os.Create(filepath.Join(subdir, "hi"))
				defer f.Close()
				cmd := exec.Command("tar", "czf", tarball, "archive")
				cmd.Dir = tempdir
				cmd.Run()
				path = fmt.Sprintf("http://%s/archive.tgz", addr)
			})

			It("returns all files with no error", func() {
				archdir := "archive"
				for i := 0; i < 3; i++ {
					name, err := r.Next()
					Expect(name).To(Equal(filepath.Join(archdir,
						fmt.Sprintf("testfile-%d", i))))
					data, _ := ioutil.ReadAll(r)
					Expect(string(data)).To(Equal(
						fmt.Sprintf("test data %d", i)))
					Expect(err).To(BeNil())
				}
				name, err := r.Next()
				Expect(name).To(Equal(filepath.Join(archdir, "zzz", "hi")))
				_, err = r.Next()
				Expect(err).To(Equal(io.EOF))
			})
		})
	})

	Describe("a file iterator", func() {
		var iterator *ArchiveIterator

		BeforeEach(func() {
			path = tempdir
			for i := 0; i < 3; i++ {
				f, _ := os.Create(filepath.Join(tempdir,
					fmt.Sprintf("testfile-%d", i)))
				defer f.Close()
				f.WriteString(fmt.Sprintf("test data %d", i))
			}
			subdir := filepath.Join(tempdir, "zzz")
			os.MkdirAll(subdir, 0777)
			f, _ := os.Create(filepath.Join(subdir, "hi"))
			defer f.Close()
		})

		JustBeforeEach(func() {
			iterator = &ArchiveIterator{Reader: &r}
		})

		It("iterates over all files with no filter", func() {
			fnames := []string{}
			for iterator.Iterate(nil) {
				fnames = append(fnames, iterator.Name())
			}
			Expect(fnames).To(HaveLen(4))
		})

		It("returns only files matching filter", func() {
			filterfunc := func(s string) bool {
				return strings.HasSuffix(s, "2")
			}
			fnames := []string{}
			for iterator.Iterate(filterfunc) {
				fnames = append(fnames, iterator.Name())
			}
			Expect(fnames).To(HaveLen(1))
		})

		It("has a nil error after successful iteration", func() {
			for iterator.Iterate(nil) {
			}
			Expect(iterator.Err()).To(BeNil())
		})

	})
})
