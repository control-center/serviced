package utils_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/zenoss/serviced/utils"

	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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

	Context("a local file", func() {
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

	Context("a local directory", func() {
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

	Context("a local tar.gz", func() {
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

	Context("a remote file", func() {
	})

	Context("a remote tar.gz", func() {
	})
})
