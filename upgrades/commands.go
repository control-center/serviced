// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package upgrades

import (
	"fmt"

	"github.com/control-center/serviced/commons"
)

type lineParser func(*parseContext, string, []string) (node, error)

var (
	nodeFactories map[string]lineParser

	DESCRIPTION = "DESCRIPTION"
	VERSION     = "VERSION"
	SNAPSHOT    = "SNAPSHOT"
	USE         = "USE"
	SVC_RUN     = "SVC_RUN"
	DEPENDENCY  = "DEPENDENCY"

	EMPTY     = "EMPTY"
	emptyNode = node{cmd: EMPTY}
)

func init() {
	nodeFactories = map[string]lineParser{
		"":          parseEmtpyCommand,
		"#":         parseEmtpyCommand,
		DESCRIPTION: atMost(1, parseDescription),
		VERSION:     atMost(1, parseOneArg),
		SNAPSHOT:    parseNoArgs,
		USE:         parseImageID,
		SVC_RUN:     parseSvcRun,
		DEPENDENCY:  validParents([]string{DESCRIPTION, VERSION}, atMost(1, parseOneArg)),
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
	return emptyNode, nil
}

func parseDescription(ctx *parseContext, cmd string, args []string) (node, error) {
	if len(args) == 0 {
		return node{}, fmt.Errorf("line %d: %s is empty", ctx.lineNum, cmd, args)
	}
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

//validParents checks that previous commands are valid or no previous commands are present
func validParents(parents []string, parser lineParser) lineParser {
	f := func(ctx *parseContext, cmd string, args []string) (node, error) {
		n, err := parser(ctx, cmd, args)
		if err == nil {
			parentMap := make(map[string]struct{})
			for _, p := range parents {
				parentMap[p] = struct{}{}
			}
			for _, previousNode := range ctx.nodes {
				if _, found := parentMap[previousNode.cmd]; !found {
					ctx.addErrorf("line %d: %s must be declared before %s", ctx.lineNum, cmd, previousNode.cmd)
				}
			}
		}
		return n, err
	}
	return f
}

// atMost checks that the command type appeart at most n times
func atMost(n int, parser lineParser) lineParser {
	f := func(ctx *parseContext, cmd string, args []string) (node, error) {

		cmdNode, err := parser(ctx, cmd, args)
		if err == nil {
			count := 0
			for _, previousNode := range ctx.nodes {
				if previousNode.cmd == cmd {
					count += 1
					if count >= n {
						ctx.addErrorf("line %d: extra %s: %s", ctx.lineNum, cmd, ctx.line)
					}
				}
			}
		}
		return cmdNode, err
	}
	return f
}
