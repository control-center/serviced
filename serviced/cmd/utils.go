package cmd

import (
	"bytes"
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

var tree_charset map[string]string
var tree_utf8 map[string]string
var tree_ascii map[string]string

func init() {
	tree_utf8 := make(map[string]string)
	tree_utf8["bar"] = "│ "
	tree_utf8["middle"] = "├─"
	tree_utf8["last"] = "└─"

	tree_ascii := make(map[string]string)
	tree_ascii["bar"] = "| "
	tree_ascii["middle"] = "|-"
	tree_ascii["last"] = "+-"

	tree_charset = tree_utf8 // default charset for tree
}

type table struct {
	writer  *tabwriter.Writer
	indent  []string
	lastrow bool
}

func newTable(minwidth, tabwidth, padding int) *table {
	w := tabwriter.NewWriter(os.Stdout, minwidth, tabwidth, padding, '\t', 0)
	return &table{writer: w}
}

func (t *table) PrintRow(columns ...interface{}) {
	fmt.Fprintf(t.writer, strings.Repeat("%v\t", len(columns)), columns...)
	fmt.Fprintln(t.writer)
}

func (t *table) PrintTreeRow(lastrow bool, columns ...interface{}) {
	t.lastrow = lastrow
	var charset = tree_charset["middle"]
	if t.lastrow {
		charset = tree_charset["last"]
	}
	columns[0] = fmt.Sprintf("%s%s%v", strings.Join(t.indent, ""), charset, columns[0])
	t.PrintRow(columns...)
}

func (t *table) Indent() {
	if t.lastrow {
		t.indent = append(t.indent, "  ")
	} else {
		t.indent = append(t.indent, tree_charset["bar"])
	}
}

func (t *table) Dedent() {
	t.indent = t.indent[:len(t.indent)-1]
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

func openEditor(data []byte, name, editor string) (reader io.Reader, err error) {
	if terminal.IsTerminal(syscall.Stdin) {
		editor, err := findEditor(editor)
		if err != nil {
			return nil, err
		}

		f, err := ioutil.TempFile("", name+"_")
		if err != nil {
			return nil, fmt.Errorf("could not open tempfile: %s", err)
		}
		defer os.Remove(f.Name())
		defer f.Close()

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

		data, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("could not read file: %s", err)
		}
		reader = bytes.NewReader(data)
	} else {
		if _, err := os.Stdout.Write(data); err != nil {
			return nil, fmt.Errorf("could not write to stdout: %s", err)
		}
		reader = os.Stdin
	}

	return reader, nil
}
