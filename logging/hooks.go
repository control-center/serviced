package logging

import (
	"fmt"
	"path"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"
)

const (
	prefix       = "github.com/control-center/serviced/"
	vendorprefix = prefix + "vendor/"
)

// ContextHook is a hook to provide context in log messages
type ContextHook struct{}

// Levels satisfies the logrus.Hook interface. This hook applies to all levels.
func (hook ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire satisfies the logrus.Hook interface. This impl figures out the file and
// line number of the caller and adds them to the data.
func (hook ContextHook) Fire(entry *logrus.Entry) error {
	pc := make([]uintptr, 3, 3)
	count := runtime.Callers(6, pc)
	for i := 0; i < count; i++ {
		fu := runtime.FuncForPC(pc[i] - 1)
		name := fu.Name()
		if strings.HasPrefix(name, prefix) && !strings.HasPrefix(name, vendorprefix) {
			file, line := fu.FileLine(pc[i] - 1)
			entry.SetField("location", fmt.Sprintf("%s:%d", path.Base(file), line))
			break
		}
	}
	return nil
}
