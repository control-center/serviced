package nfs

import (
	"testing"
)

type mockCommand struct {
	name   string
	args   []string
	output []byte
	err    error
}

func (c *mockCommand) CombinedOutput() ([]byte, error) {
	return c.output, c.err
}

type mountTestCaseT struct {
	remote   string
	local    string
	expected error
}

var mountTestCases = []mountTestCaseT{
	mountTestCaseT{"127.0.0.1:/tmp", "/test", nil},
	mountTestCaseT{"127.0sf1:/tmp", "/test", ErrMalformedNFSMountpoint},
	mountTestCaseT{"127.0.0.1:tmp", "/test", ErrMalformedNFSMountpoint},
}

func TestMount(t *testing.T) {

	// save current command factory to stack for later restoration
	defer func(c func(string, ...string) command) {
		commandFactory = c
	}(commandFactory)

	commandFactory = func(name string, args ...string) command {
		return &mockCommand{
			name: name,
			args: args,
		}
	}

	for _, testcase := range mountTestCases {
		if err := Mount(testcase.remote, testcase.local); err != testcase.expected {
			t.Fatalf("failed on testcase: %+v, got %s", testcase, err)
		}
	}
}
