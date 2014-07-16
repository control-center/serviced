// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package subprocess

import (
	"fmt"
	"testing"

	"github.com/zenoss/serviced/utils"
)

func TestExecutorCapturesStdout(t *testing.T) {
	executor := Executor{}
	for i := 0; i <= 5; i++ {
		executor.Submit("echo", fmt.Sprintf("%d", i))
	}

	executor.Execute()
	for i := 0; i <= 5; i++ {
		if executor.Results[i].Error != nil {
			t.Fatalf("Result[%d].Error: nil != %s", i, executor.Results[i].Error)
		}

		expected := fmt.Sprintf("%d\n", i)
		if executor.Results[i].Stdout.String() != expected {
			t.Fatalf("Result[%d].Stdout: \"%s\" != \"%s\"", i, expected, executor.Results[i].Stdout.String())
		}

		if executor.Results[i].Stderr.String() != "" {
			t.Fatalf("Result[%d].Stderr: \"\" != \"%s\"", i, executor.Results[i].Stderr.String())
		}
	}
}

func TestExecutorCapturesStderr(t *testing.T) {
	executor := Executor{}
	for i := 0; i <= 5; i++ {
		executor.Submit("python", "-c", fmt.Sprintf("import sys; print >> sys.stderr, %d", i))
	}

	executor.Execute()
	for i := 0; i <= 5; i++ {
		if executor.Results[i].Error != nil {
			t.Fatalf("Result[%d].Error: nil != %s", i, executor.Results[i].Error)
		}

		if executor.Results[i].Stdout.String() != "" {
			t.Fatalf("Result[%d].Stdout: \"\" != \"%s\"", i, executor.Results[i].Stdout.String())
		}

		expected := fmt.Sprintf("%d\n", i)
		if executor.Results[i].Stderr.String() != expected {
			t.Fatalf("Result[%d].Stderr: \"%s\" != \"%s\"", i, expected, executor.Results[i].Stderr.String())
		}
	}
}

func TestExecutorCapturesError(t *testing.T) {
	executor := Executor{}
	for i := 0; i <= 5; i++ {
		executor.Submit("python", "-c", fmt.Sprintf("import sys; sys.exit(%d)", i))
	}

	executor.Execute()
	for i := 0; i <= 5; i++ {
		if status, ok := utils.GetExitStatus(executor.Results[i].Error); !ok || status != i {
			t.Fatalf("Result[%d].Error:%s (%d,%t) != (%d,true)", i, executor.Results[i].Error, status, ok, i)
		}

		if executor.Results[i].Stdout.String() != "" {
			t.Fatalf("Result[%d].Stdout: \"\" != \"%s\"", i, executor.Results[i].Stdout.String())
		}

		if executor.Results[i].Stderr.String() != "" {
			t.Fatalf("Result[%d].Stderr: \"\" != \"%s\"", i, executor.Results[i].Stderr.String())
		}
	}
}
