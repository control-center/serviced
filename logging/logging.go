package logging

import (
	"runtime"
	"strings"

	"github.com/zenoss/logri"
)

func init() {
	logri.AddHook(ContextHook{})
}

func pkgFromFunc(funcname string) string {
	subpkg := strings.TrimPrefix(funcname, prefix)
	parts := strings.Split(subpkg, ".")
	pkg := ""
	if parts[len(parts)-2] == "(" {
		pkg = strings.Join(parts[0:len(parts)-2], ".")
	} else {
		pkg = strings.Join(parts[0:len(parts)-1], ".")
	}
	return strings.Replace(pkg, "/", ".", -1)
}

// PackageLogger returns a logger for a given package.
func PackageLogger() *logri.Logger {
	pc := make([]uintptr, 3, 3)
	count := runtime.Callers(2, pc)
	for i := 0; i < count; i++ {
		fu := runtime.FuncForPC(pc[i])
		name := fu.Name()
		if strings.Contains(name, prefix) {
			return logri.GetLogger(pkgFromFunc(name))
		}
	}
	return logri.GetLogger("")
}
