// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package upgrades

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/control-center/serviced/commons"
)

var (
	commandFactories map[string]func(*parseContext, []string) (command, error)
	emptyCMD         = emptyCommand("")
)

func init() {
	commandFactories = map[string]func(*parseContext, []string) (command, error){
		"":            newEmtpyCommand,
		"#":           newComment,
		"DESCRIPTION": newDescription,
		"VERSION":     newVersion,
		"SNAPSHOT":    newSnapshot,
		"USE":         newUse,
		"SVC_RUN":     newSvcRun,
		"DEPENDENCY":  newDependency,
	}
}

func newEmtpyCommand(ctx *parseContext, args []string) (command, error) {
	if strings.TrimLeftFunc(ctx.line, unicode.IsSpace) != "" {
		return nil, fmt.Errorf("expected empty line, got: %s", ctx.line)
	}
	if len(args) != 0 {
		return nil, fmt.Errorf("expected empty args, got: %s", ctx.line)
	}
	return emptyCMD, nil
}

func newComment(ctx *parseContext, args []string) (command, error) {
	if !strings.HasPrefix(strings.TrimLeftFunc(ctx.line, unicode.IsSpace), "#") {
		return nil, fmt.Errorf("expected comment line, got: %s", ctx.line)
	}

	return comment(ctx.line), nil
}

func newDescription(ctx *parseContext, args []string) (command, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("DESCRIPTION is empty", args)
	}
	if ctx.descriptionCount > 0 {
		ctx.addErrorf("Extra DESCRIPTION at line %d: %s", ctx.lineNum, ctx.line)
	}
	ctx.descriptionCount += 1
	desc := strings.TrimFunc(strings.Trim(strings.TrimLeftFunc(ctx.line, unicode.IsSpace), "DESCRIPTION"), unicode.IsSpace)
	return description(desc), nil
}

func newVersion(ctx *parseContext, args []string) (command, error) {
	if len(args) == 0 || len(args) > 1 {
		return nil, fmt.Errorf("expected one argument, got: %s", ctx.line)
	}
	return version(args[0]), nil
}

func newSnapshot(ctx *parseContext, args []string) (command, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("SNAPSHOT does not accept arguments: %s", ctx.line)
	}
	return snapshot(ctx.line), nil
}
func newUse(ctx *parseContext, args []string) (command, error) {
	if len(args) == 0 || len(args) > 1 {
		return nil, fmt.Errorf("expected one argument, got: %s", ctx.line)
	}
	return createUse(args[0])
}

func createUse(image string) (use, error) {
	imageID, err := commons.ParseImageID(image)
	if err != nil {
		return use{}, err
	}
	return use{*imageID}, nil
}

func newSvcRun(ctx *parseContext, args []string) (command, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("expected at least two arguments, got: %s", ctx.line)
	}
	//TODO: validate contents
	return svc_run{args[0], args[1], args[2:len(args)]}, nil
}

func newDependency(ctx *parseContext, args []string) (command, error) {
	if len(args) == 0 || len(args) > 1 {
		return nil, fmt.Errorf("expected one argument, got: %s", ctx.line)
	}
	//TODO: verify content format
	return dependency(args[0]), nil
}

type emptyCommand string

// comment starts with #
type comment string

// DEPENDS serviced_version
type dependency string

//DESCRIPTION  Zenoss RM 5.0.1 upgrade
type description string

//VERSION   resmgr-5.0.1
type version string

//SNAPSHOT
type snapshot string

//USE  zenoss/resmgr-stable:5.0.1
type use struct {
	image commons.ImageID
}

//SVC_RUN  /zope upgrade
type svc_run struct {
	container string
	command   string
	args      []string
}
