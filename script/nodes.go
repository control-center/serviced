// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/control-center/serviced/commons"
)

var (
	nodeFactories map[string]lineParser

	DESCRIPTION = "DESCRIPTION"
	VERSION     = "VERSION"
	SNAPSHOT    = "SNAPSHOT"
	REQUIRE_SVC = "REQUIRE_SVC"
	USE         = "SVC_USE"
	SVC_RUN     = "SVC_RUN"
	SVC_EXEC    = "SVC_EXEC"
	SVC_START   = "SVC_START"
	SVC_STOP    = "SVC_STOP"
	SVC_RESTART = "SVC_RESTART"
	SVC_WAIT    = "SVC_WAIT"
	SVC_MIGRATE = "SVC_MIGRATE"
	DEPENDENCY  = "DEPENDENCY"

	EMPTY     = "EMPTY"
	emptyNode = node{cmd: EMPTY}
)

func init() {
	nodeFactories = map[string]lineParser{
		"":          parseEmptyCommand,
		"#":         parseEmptyCommand,
		DESCRIPTION: atMost(1, parseArgCount(min(1), buildNode)),
		VERSION:     atMost(1, parseArgCount(equals(1), buildNode)),
		REQUIRE_SVC: atMost(1, parseArgCount(equals(0), buildNode)),
		SNAPSHOT:    require([]string{REQUIRE_SVC}, parseArgCount(max(1), buildNode)),
		USE:         require([]string{REQUIRE_SVC}, parseImageID(parseArgCount(equals(1), buildNode))),
		SVC_RUN:     require([]string{REQUIRE_SVC}, parseArgCount(min(2), buildNode)),
		// eg., SVC_EXEC NO_COMMIT Zenoss.core/Zope /run/my/script.sh --arg1 arg2
		SVC_EXEC:    require([]string{REQUIRE_SVC}, parseArgMatch(0, "^(NO_)?COMMIT$", false, parseArgCount(min(3), buildNode))),
		SVC_START:   require([]string{REQUIRE_SVC}, parseArgMatch(1, "^recurse$|^auto$", true, parseArgCount(bounds(1, 2), buildNode))),
		SVC_RESTART: require([]string{REQUIRE_SVC}, parseArgMatch(1, "^recurse$|^auto$", true, parseArgCount(bounds(1, 2), buildNode))),
		SVC_STOP:    require([]string{REQUIRE_SVC}, parseArgMatch(1, "^recurse$|^auto$", true, parseArgCount(bounds(1, 2), buildNode))),
		SVC_WAIT:    parseWait,
		SVC_MIGRATE: require([]string{REQUIRE_SVC}, parseArgCount(bounds(1, 2), parseSDKVersion(buildNode))),
		DEPENDENCY:  validParents([]string{DESCRIPTION, VERSION}, atMost(1, parseArgCount(equals(1), buildNode))),
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

type lineParser func(*parseContext, string, []string) (node, error)

type match func(int) error

func equals(n int) match {
	return func(x int) error {
		if x != n {
			return fmt.Errorf("expected %v, got %v", n, x)
		}
		return nil
	}
}

func bounds(minN, maxN int) match {
	return func(x int) error {
		if err := min(minN)(x); err != nil {
			return err
		}
		if err := max(maxN)(x); err != nil {
			return err
		}
		return nil
	}
}

func max(n int) match {
	return func(x int) error {
		if n >= x {
			return nil
		}
		return fmt.Errorf("expected at most %v, got %v", n, x)
	}
}

func min(n int) match {
	return func(x int) error {
		if n <= x {
			return nil
		}
		return fmt.Errorf("expected at least %v, got %v", n, x)
	}
}

func parseEmptyCommand(ctx *parseContext, cmd string, args []string) (node, error) {
	return emptyNode, nil
}

func buildNode(ctx *parseContext, cmd string, args []string) (node, error) {
	return node{cmd: cmd, line: ctx.line, lineNum: ctx.lineNum, args: args}, nil
}

func parseSDKVersion(parser lineParser) lineParser {
	f := func(ctx *parseContext, cmd string, args []string) (node, error) {
		n, err := parser(ctx, cmd, args)
		if err == nil {
			if len(args) == 2 {
				if _, err := findSDKVersion(args[0]); err != nil {
					return node{}, fmt.Errorf("line %d: %v", ctx.lineNum, err)
				}
			}
		}
		return n, err
	}
	return f
}

func findSDKVersion(arg string) (version string, err error) {
	//try to match
	const SDK_VERSION_PATTERN = `^SDK=([a-zA-Z0-9.\-_]+)$`
	sdkVerRegex := regexp.MustCompile(SDK_VERSION_PATTERN)
	matches := sdkVerRegex.FindStringSubmatch(arg)
	if len(matches) != 2 || matches[1] == "" {
		return "", fmt.Errorf("arg %s did not match %s", arg, SDK_VERSION_PATTERN)
	}
	return matches[1], nil
}

func parseArgMatch(argN int, pattern string, optional bool, parser lineParser) lineParser {
	f := func(ctx *parseContext, cmd string, args []string) (node, error) {
		n, err := parser(ctx, cmd, args)
		if err == nil {
			if argN < len(args) {
				//try to match
				if matched, err := regexp.MatchString(pattern, args[argN]); !matched {
					return node{}, fmt.Errorf("line %d: arg %s did not match %s", ctx.lineNum, args[argN], pattern)
				} else if err != nil {
					return node{}, fmt.Errorf("line %d: %v", ctx.lineNum, err)
				}
			} else if !optional {
				return node{}, fmt.Errorf("line %d: no arg at position %v", ctx.lineNum, argN)
			}
		}
		return n, err
	}
	return f
}

func parseArgCount(matcher match, parser lineParser) lineParser {
	f := func(ctx *parseContext, cmd string, args []string) (node, error) {
		n, err := parser(ctx, cmd, args)
		if err == nil {
			if err := matcher(len(args)); err != nil {
				return node{}, fmt.Errorf("line %d: %v: %s", ctx.lineNum, err, ctx.line)
			}
		}
		return n, err
	}
	return f
}
func parseImageID(parser lineParser) lineParser {
	return func(ctx *parseContext, cmd string, args []string) (node, error) {
		n, err := parser(ctx, cmd, args)
		if err == nil {
			_, err := commons.ParseImageID(args[0])
			if err != nil {
				return node{}, err
			}
		}
		return n, err
	}
}

//validParents checks that there are no previous command or previous commands are only in parents list
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

//require checks required commands are already present
func require(required []string, parser lineParser) lineParser {
	f := func(ctx *parseContext, cmd string, args []string) (node, error) {
		n, err := parser(ctx, cmd, args)
		if err == nil {
			requiredMap := make(map[string]bool)
			for _, r := range required {
				requiredMap[r] = false //hasn't been found yet
			}

			for _, previousNode := range ctx.nodes {
				if _, found := requiredMap[previousNode.cmd]; found {
					requiredMap[previousNode.cmd] = true
				}
			}
			for key, found := range requiredMap {
				if !found {
					ctx.addErrorf("line %d: %s depends on %s", ctx.lineNum, cmd, key)
				}
			}
		}
		return n, err
	}
	return f
}

// atMost checks that the command type appears at most n times
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

// SVC_WAIT <service_id>+ (started|stopped|paused) <timeout>?
func parseWait(ctx *parseContext, cmd string, args []string) (node, error) {
	stateIdx := -1
	for i, arg := range args {
		if arg == "started" || arg == "stopped" || arg == "paused" {
			stateIdx = i
			break
		}
	}

	var err error
	switch {
	case stateIdx == -1:
		err = fmt.Errorf("line %d: missing state argument (started|stopped|paused)", ctx.lineNum)
	case stateIdx == 0:
		err = fmt.Errorf("line %d: missing service id", ctx.lineNum)
	case stateIdx < len(args)-2:
		err = fmt.Errorf("line %d: too many arguments following state: %v", ctx.lineNum, args[stateIdx:])
	case stateIdx == len(args)-2:
		if strings.Trim(args[len(args)-1], "1234567890") != "" {
			err = fmt.Errorf("line %d: expected integer timeout; got %s", ctx.lineNum, args[len(args)-1])
		}
	}

	if err != nil {
		return node{}, err
	}
	return node{cmd: cmd, line: ctx.line, lineNum: ctx.lineNum, args: args}, nil
}
