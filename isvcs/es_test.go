package isvcs

import (
	"testing"
)

func TestPurge(t *testing.T) {
	Init()
	Mgr.Start()
	PurgeLogstashIndices(10, 10)
}
