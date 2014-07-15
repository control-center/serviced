// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"

	"code.google.com/p/go.crypto/ssh/terminal"
)

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
