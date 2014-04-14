package zk_driver

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	zklib "github.com/samuel/go-zookeeper/zk"
)

var (
	ErrDeadlock      = errors.New("zk: trying to acquire a lock twice")
	ErrNotLocked     = errors.New("zk: not locked")
	ErrNoLeaderFound = errors.New("zk: no leader found")
)

type Leader struct {
	c        *Connection
	path     string
	lockPath string
	seq      uint64
	data     []byte
}

func parseSeq(path string) (uint64, error) {
	parts := strings.Split(path, "-")
	return strconv.ParseUint(parts[len(parts)-1], 10, 64)
}

func (l *Leader) prefix() string {
	return fmt.Sprintf("%s/leader-", l.path)
}

func (l *Leader) Current() (data []byte, err error) {

	children, _, err := l.c.conn.Children(l.path)
	if err != nil {
		return data, err
	}

	var lowestSeq uint64
	lowestSeq = math.MaxUint64
	path := ""
	for _, p := range children {
		s, err := parseSeq(p)
		if err != nil {
			return data, err
		}
		if s < lowestSeq {
			lowestSeq = s
			path = p
		}
	}
	if lowestSeq == math.MaxUint64 {
		return data, ErrNoLeaderFound
	}
	path = fmt.Sprintf("%s/%s", l.path, path)
	data, _, err = l.c.conn.Get(path)
	return data, err
}

func (l *Leader) TakeLead() error {
	if l.lockPath != "" {
		return ErrDeadlock
	}

	prefix := l.prefix()

	path := ""
	var err error
	for i := 0; i < 3; i++ {
		path, err = l.c.conn.CreateProtectedEphemeralSequential(prefix, l.data, zklib.WorldACL(zklib.PermAll))
		if err == zklib.ErrNoNode {
			// Create parent node.
			parts := strings.Split(l.path, "/")
			pth := ""
			for _, p := range parts[1:] {
				pth += "/" + p
				_, err := l.c.conn.Create(pth, []byte{}, 0, zklib.WorldACL(zklib.PermAll))
				if err != nil && err != zklib.ErrNodeExists {
					return err
				}
			}
		} else if err == nil {
			break
		} else {
			return err
		}
	}
	if err != nil {
		return err
	}
	seq, err := parseSeq(path)
	if err != nil {
		return err
	}

	for {
		children, _, err := l.c.conn.Children(l.path)
		if err != nil {
			return err
		}

		lowestSeq := seq
		var prevSeq uint64
		prevSeqPath := ""
		for _, p := range children {
			s, err := parseSeq(p)
			if err != nil {
				return err
			}
			if s < lowestSeq {
				lowestSeq = s
			}
			if s < seq && s > prevSeq {
				prevSeq = s
				prevSeqPath = p
			}
		}

		if seq == lowestSeq {
			// Acquired the lock
			break
		}

		// Wait on the node next in line for the lock
		_, _, ch, err := l.c.conn.GetW(l.path + "/" + prevSeqPath)
		if err != nil && err != zklib.ErrNoNode {
			return err
		} else if err != nil && err == zklib.ErrNoNode {
			// try again
			continue
		}

		ev := <-ch
		if ev.Err != nil {
			return ev.Err
		}
	}

	l.seq = seq
	l.lockPath = path
	return nil
}

func (l *Leader) ReleaseLead() error {
	if l.lockPath == "" {
		return ErrNotLocked
	}
	if err := l.c.conn.Delete(l.lockPath, -1); err != nil {
		return err
	}
	l.lockPath = ""
	l.seq = 0
	return nil
}
