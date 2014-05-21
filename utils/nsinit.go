package utils

import (
	"io/ioutil"
	"os"
	"syscall"
)

var BASH_SCRIPT = `
DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMAND="$@"
trap "rm -f ${DIR}/$$.stderr" EXIT
for i in {1..10}; do
	SEEN=0
	${COMMAND} 2> >(tee ${DIR}/$$.stderr >&2)
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
	defer os.Remove(f.Name())
	if _, err := f.WriteString(BASH_SCRIPT); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	command := []string{f.Name()}
	command = append(command, cmd...)
	err = syscall.Exec("/bin/bash", command, os.Environ())
	return nil
}