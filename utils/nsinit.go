package utils

import (
	"io/ioutil"
	"os"
	"strings"
	"syscall"

	"github.com/zenoss/glog"
)

var BASH_SCRIPT = `
DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
#trap "rm -f ${DIR}/$$.stderr" EXIT
for i in {1..10}; do
	SEEN=0
	{{{{COMMAND}}}} 2> >(tee ${DIR}/$$.stderr >&2)
	RESULT=$?
	[ "${RESULT}" == 0 ] && exit 0
	grep setns ${DIR}/$$.stderr || exit ${RESULT}
done
exit ${RESULT}
`

func NSInitWithRetry(cmd []string) error {
	f, err := ioutil.TempFile("", "nsinit")
	if err != nil {
		return err
	}
	defer f.Close()
	//defer os.Remove(f.Name())
	script := strings.Replace(BASH_SCRIPT, "{{{{COMMAND}}}}", strings.Join(cmd, " "), 1)
	if _, err := f.WriteString(script); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	command := []string{f.Name()}
	glog.V(0).Infof("Here's the command: %s", command)
	err = syscall.Exec("/bin/bash", command, os.Environ())
	return nil
}