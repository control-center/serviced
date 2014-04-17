package cmd

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/tabwriter"

	"code.google.com/p/go.crypto/ssh/terminal"
)

type table struct {
	writer *tabwriter.Writer
}

func newTable(minwidth, tabwidth, padding int) *table {
	w := tabwriter.NewWriter(os.Stdout, minwidth, tabwidth, padding, '\t', 0)
	return &table{w}
}

func (t *table) PrintRow(columns ...interface{}) {
	fmt.Fprintf(t.writer, strings.Repeat("%v\t", len(columns)), columns...)
	fmt.Fprintln(t.writer)
}

func (t *table) Flush() {
	t.writer.Flush()
}

func remove(index int, list ...interface{}) []interface{} {
	var (
		left  []interface{}
		right []interface{}
	)

	switch {
	case index < 0 || index > len(list):
		panic("index out of bounds")
	case index+1 < len(list):
		right = list[index+1:]
		fallthrough
	default:
		left = list[:index]
	}

	return append(left, right...)
}

var editors = []string{"vim", "vi", "nano"}

func findEditor(editor string) (string, error) {
	if editor != "" {
		path, err := exec.LookPath(editor)
		if err != nil {
			return "", fmt.Errorf("editor (%s) not found: %s", editor, err)
		}
		return path, nil
	}
	for _, e := range editors {
		path, err := exec.LookPath(e)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no editor found")
}

func openEditor(data []byte, name, editor string) (io.Reader, error) {
	var reader io.Reader
	if terminal.IsTerminal(syscall.Stdin) {
		editor, err := findEditor(editor)
		if err != nil {
			return nil, err
		}

		f, err := ioutil.TempFile("", name+"_")
		if err != nil {
			return nil, fmt.Errorf("could not open tempfile: %s", err)
		}
		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()

		if _, err := f.Write(data); err != nil {
			return nil, fmt.Errorf("could not write tempfile: %s", err)
		}

		e := exec.Command(editor, f.Name())
		e.Stdin = os.Stdin
		e.Stdout = os.Stdout
		e.Stderr = os.Stderr

		if err := e.Run(); err != nil {
			return nil, fmt.Errorf("received error from editor: %s (%s)", err, editor)
		}
		if _, err := f.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("could not seek file: %s", err)
		}

		reader = bufio.NewReader(f)
	} else {
		if _, err := os.Stdout.Write(data); err != nil {
			return nil, fmt.Errorf("could not write to stdout: %s", err)
		}
		reader = os.Stdin
	}

	return reader, nil
}
