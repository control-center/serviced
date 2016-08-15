package logging

import (
	"path"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"
)

const (
	prefix = "github.com/control-center/serviced/"
)

type ContextHook struct{}

func (hook ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook ContextHook) Fire(entry *logrus.Entry) error {
	pc := make([]uintptr, 3, 3)
	count := runtime.Callers(6, pc)
	for i := 0; i < count; i++ {
		fu := runtime.FuncForPC(pc[i] - 1)
		name := fu.Name()
		if strings.Contains(name, prefix) {
			file, line := fu.FileLine(pc[i] - 1)
			entry.Data["file"] = path.Base(file)
			entry.Data["line"] = line
			break
		}
	}
	return nil
}
