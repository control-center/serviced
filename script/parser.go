// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/mattn/go-shellwords"
)

type parseContext struct {
	lineNum int
	line    string
	errors  []error
	nodes   []node
}

func newParseContext() *parseContext {
	return &parseContext{errors: []error{}, nodes: []node{}}
}

func (pc *parseContext) addErrorf(format string, a ...interface{}) {
	pc.errors = append(pc.errors, fmt.Errorf(format, a...))
}

func parseFile(filePath string) (*parseContext, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)
	return parseDescriptor(r)
}

func parseDescriptor(r io.Reader) (*parseContext, error) {
	ctx := newParseContext()
	parse := func(num int, line string) error {
		ctx.lineNum = num
		ctx.line = line
		return parseNode(ctx)
	}
	if err := ForEachLine(r, parse); err != nil {
		return nil, err
	}
	return ctx, nil
}

func ForEachLine(r io.Reader, apply func(num int, line string) error) error {
	scanner := bufio.NewScanner(r)
	i := 0
	for scanner.Scan() {
		i += 1
		if err := apply(i, scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}

//parseLine returns the command and args array if any. If line is empty empty string and args are returned
func parseLine(line string) (string, []string, error) {
	line = strings.TrimLeftFunc(line, unicode.IsSpace)
	shellwords.ParseEnv = false
	shellwords.ParseBacktick = false
	fields, err := shellwords.Parse(line)
	if err != nil {
		return "", []string{}, err
	}

	//ignore empty lines
	if len(fields) == 0 {
		return "", []string{}, nil
	}
	var cmd string
	if strings.HasPrefix(fields[0], "#") {
		cmd = "#"
	} else {
		cmd = fields[0]
	}
	args := fields[1:] //remove first element(cmd)
	for i, _ := range args {
		args[i] = strings.TrimFunc(args[i], unicode.IsSpace)
	}
	return cmd, args, nil
}

// parseCommand parses current line and creates a command
func parseNode(ctx *parseContext) error {
	prefix, args, err := parseLine(ctx.line)
	if err != nil {
		return fmt.Errorf("could not parse line %d: %s: %s", ctx.lineNum, ctx.line, err)
	}

	f, found := nodeFactories[prefix]
	if !found {
		return fmt.Errorf("could not parse line %d: %s", ctx.lineNum, ctx.line)
	}
	node, err := f(ctx, prefix, args)
	if err != nil {
		return err
	}
	if node.cmd != EMPTY {
		ctx.nodes = append(ctx.nodes, node)
	}
	return nil
}
