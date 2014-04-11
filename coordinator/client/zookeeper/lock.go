package zk_driver

import (
	"fmt"
	"log"
	"os"
	"strings"
	"strconv"

	zklib "github.com/samuel/go-zookeeper/zk"
)

type Lock struct {
	conn *Connection
	path string
	guid string
	lockId int
}

func (l *Lock) Lock() (err error) {
	parts := strings.Split(l.path, "/")
	currentPath := ""
	for i, part := range parts {
		if i == 0 {
			continue
		}
		currentPath += "/" + part

		err = l.conn.CreateDir(currentPath)
		if err == zklib.ErrNodeExists {
			continue
		}
	}
	lockPath := currentPath + "/" + l.guid + "_"
	lockPathKey, err := l.conn.conn.Create(lockPath, []byte{}, zklib.FlagEphemeral+zklib.FlagSequence, zklib.WorldACL(zklib.PermAll))
	if err != nil {
		return err
	}
	parts = strings.Split(lockPathKey, "/")
	lockParts := strings.Split(parts[len(parts)-1], "_")
	if len(lockParts) != 2 {
		panic(fmt.Errorf("unknown format of lockparts: %v", lockParts))
	}
	lockId, err := strconv.Atoi(lockParts[1])
	if err != nil {
		return err
	}
	l.lockId = lockId
	children, _, err := l.conn.conn.Children(currentPath)
	log.Printf("children %s, this one: %s:%d", children, l.guid, l.lockId)
	return ErrUnimplemented
}

func (l *Lock) Unlock() error {
	return ErrUnimplemented
}

var urandomFilename = "/dev/urandom"

// Generate a new UUID
func newUuid() (string, error) {
	f, err := os.Open(urandomFilename)
	if err != nil {
		return "", err
	}
	b := make([]byte, 16)
	defer f.Close()
	f.Read(b)
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid, err
}
