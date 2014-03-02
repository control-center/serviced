package utils_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/zenoss/serviced/utils"

	"fmt"
	"io/ioutil"
	"os"
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
				_, ok := err.(*EndOfArchive)
				Expect(ok).To(BeTrue())
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
						fmt.Sprintf("testfile-%s", i)))
					defer f.Close()
					f.WriteString(fmt.Sprintf("test data %s", i))
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
						fmt.Sprintf("testfile-%s", i))))
					data, _ := ioutil.ReadAll(r)
					Expect(string(data)).To(Equal(
						fmt.Sprintf("test data %s", i)))
					Expect(err).To(BeNil())
				}
				name, err := r.Next()
				Expect(name).To(Equal(filepath.Join(tempdir, "zzz", "hi")))
				_, err = r.Next()
				_, ok := err.(*EndOfArchive)
				Expect(ok).To(BeTrue())
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
	})

	Context("a remote file", func() {
	})

	Context("a remote tar.gz", func() {
	})
})
