// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package upgrades

import (
	"fmt"

	"github.com/control-center/serviced/commons"
)

var (
	nodeFactories map[string]func(*parseContext, string, []string) (node, error)

	DESCRIPTION = "DESCRIPTION"
	VERSION     = "VERSION"
	SNAPSHOT    = "SNAPSHOT"
	USE         = "USE"
	SVC_RUN     = "SVC_RUN"
	DEPENDENCY  = "DEPENDENCY"
)

func init() {
	nodeFactories = map[string]func(*parseContext, string, []string) (node, error){
		"":          parseEmtpyCommand,
		"#":         parseEmtpyCommand,
		DESCRIPTION: parseDescription,
		VERSION:     parseOneArg,
		SNAPSHOT:    parseNoArgs,
		USE:         parseImageID,
		SVC_RUN:     parseSvcRun,
		DEPENDENCY:  parseOneArg,
	}
}

// node is the struct created from parsing a line; cmd is the command on the line, args are the remainder of the line, line is
// the original line and lineNum is the line number where the line occurred.
type node struct {
	cmd     string
	args    []string
	line    string
	lineNum int
}

func parseEmtpyCommand(ctx *parseContext, cmd string, args []string) (node, error) {
	return node{line: ctx.line, lineNum: ctx.lineNum, args: []string{}}, nil
}

func parseDescription(ctx *parseContext, cmd string, args []string) (node, error) {
	if len(args) == 0 {
		return node{}, fmt.Errorf("line %d: %s is empty", ctx.lineNum, cmd, args)
	}
	if ctx.descriptionCount > 0 {
		ctx.addErrorf("line %d: extra %s: %s", ctx.lineNum, cmd, ctx.line)
	}
	ctx.descriptionCount += 1
	return node{cmd: cmd, line: ctx.line, lineNum: ctx.lineNum, args: args}, nil
}

func parseOneArg(ctx *parseContext, cmd string, args []string) (node, error) {
	if len(args) != 1 {
		return node{}, fmt.Errorf("line %d: expected one argument, got: %s", ctx.lineNum, ctx.line)
	}
	return node{cmd: cmd, line: ctx.line, lineNum: ctx.lineNum, args: args}, nil
}

func parseNoArgs(ctx *parseContext, cmd string, args []string) (node, error) {
	if len(args) != 0 {
		return node{}, fmt.Errorf("line %d: %s does not accept arguments: %s", ctx.lineNum, cmd, ctx.line)
	}
	return node{cmd: cmd, line: ctx.line, lineNum: ctx.lineNum, args: []string{}}, nil
}

func parseImageID(ctx *parseContext, cmd string, args []string) (node, error) {
	if len(args) != 1 {
		return node{}, fmt.Errorf("line %d: expected one argument, got: %s", ctx.lineNum, ctx.line)
	}
	_, err := commons.ParseImageID(args[0])
	if err != nil {
		return node{}, err
	}

	return node{cmd: cmd, line: ctx.line, lineNum: ctx.lineNum, args: args}, nil
}

func parseSvcRun(ctx *parseContext, cmd string, args []string) (node, error) {
	if len(args) < 2 {
		return node{}, fmt.Errorf("line %d: expected at least two arguments, got: %s", ctx.lineNum, ctx.line)
	}
	//TODO: validate contents
	return node{cmd: cmd, line: ctx.line, lineNum: ctx.lineNum, args: args}, nil
}
