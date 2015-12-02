# coding: latin-1

# Copyright 2015 The Serviced Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import unittest

import gostack
from utils import populateGoroutine

# Possible formats:
#   goroutine 0 [idle]:
#   goroutine 1 [chan receive, 30 minutes]:
#   goroutine 17 [syscall, 31 minutes, locked to thread]:
#   goroutine 34 [syscall, locked to thread]:

class BaseTestCase(unittest.TestCase):
    def setUp(self):
        self.goroutine = gostack.Goroutine()

class EqualityBaseTestCase(BaseTestCase):
    def setUp(self):
        super(EqualityBaseTestCase, self).setUp()
        self.goroutine2 = gostack.Goroutine()

#-------------------------------------------
# Parsing

class Goroutine_GoodWithoutWaitTime(BaseTestCase):
    def runTest(self):
        warnings = self.goroutine.parseLine('goroutine 6 [chan receive]:')
        self.assertEqual(self.goroutine.id, 6)
        self.assertEqual(self.goroutine.state, 'chan receive')
        self.assertEqual(self.goroutine.waittime, 0)
        self.assertFalse(self.goroutine.lockedtothread)
        self.assertEqual(len(self.goroutine.frames), 0)
        self.assertEqual(len(warnings), 0)

class Goroutine_GoodWithWaitTime(BaseTestCase):
    def runTest(self):
        warnings = self.goroutine.parseLine('goroutine 12 [select, 30 minutes]:')
        self.assertEqual(self.goroutine.id, 12)
        self.assertEqual(self.goroutine.state, 'select')
        self.assertEqual(self.goroutine.waittime, 30)
        self.assertFalse(self.goroutine.lockedtothread)
        self.assertEqual(len(self.goroutine.frames), 0)
        self.assertEqual(len(warnings), 0)

class Goroutine_GoodLockedToThreadWithoutWaitTime(BaseTestCase):
    def runTest(self):
        warnings = self.goroutine.parseLine('goroutine 1034 [syscall, locked to thread]:')
        self.assertEqual(self.goroutine.id, 1034)
        self.assertEqual(self.goroutine.state, 'syscall')
        self.assertEqual(self.goroutine.waittime, 0)
        self.assertTrue(self.goroutine.lockedtothread)
        self.assertEqual(len(self.goroutine.frames), 0)
        self.assertEqual(len(warnings), 0)

class Goroutine_GoodLockedToThreadWithWaitTime(BaseTestCase):
    def runTest(self):
        warnings = self.goroutine.parseLine('goroutine 5555 [IO wait, 240 minutes, locked to thread]:')
        self.assertEqual(self.goroutine.id, 5555)
        self.assertEqual(self.goroutine.state, 'IO wait')
        self.assertEqual(self.goroutine.waittime, 240)
        self.assertTrue(self.goroutine.lockedtothread)
        self.assertEqual(len(self.goroutine.frames), 0)
        self.assertEqual(len(warnings), 0)

class Goroutine_BadNotGoroutine(BaseTestCase):
    def runTest(self):
        with self.assertRaises(gostack.ParseError):
            self.goroutine.parseLine('blah 12 [select, 30 minutes]:')

class Goroutine_BadNonNumericId(BaseTestCase):
    def runTest(self):
        with self.assertRaises(gostack.ParseError):
            self.goroutine.parseLine('goroutine #49 [chan receive]:')

class Goroutine_BadNoStateWaittime(BaseTestCase):
    def runTest(self):
        with self.assertRaises(gostack.ParseError):
            self.goroutine.parseLine('goroutine 22 something else here:')

class Goroutine_WarnNotMinutes(BaseTestCase):
    def runTest(self):
        warnings = self.goroutine.parseLine('goroutine 12 [select, 30 hours]:')
        self.assertEqual(len(warnings), 1)
        self.assertTrue(warnings[0].lower().find('unknown field'))

class Goroutine_WarnExtraStateInfoField(BaseTestCase):
    def runTest(self):
        warnings = self.goroutine.parseLine('goroutine 8 [chan send, 30 minutes, locked to thread, other]:')
        self.assertEqual(self.goroutine.id, 8)
        self.assertEqual(self.goroutine.state, 'chan send')
        self.assertEqual(self.goroutine.waittime, 30)
        self.assertTrue(self.goroutine.lockedtothread)
        self.assertEqual(len(self.goroutine.frames), 0)
        self.assertEqual(len(warnings), 1)
        self.assertTrue(warnings[0].lower().find('unknown field'))

#-------------------------------------------
# Equality

class Goroutine_EqualNoStackFrames(EqualityBaseTestCase):
    def runTest(self):
        # All other attributes are different
        warnings = self.goroutine.parseLine('goroutine 15 [select, 199 minutes]:')
        self.assertEqual(len(warnings), 0)

        warnings = self.goroutine2.parseLine('goroutine 24829 [IO wait]:')
        self.assertEqual(len(warnings), 0)

        self.assertTrue(self.goroutine == self.goroutine2)
        self.assertFalse(self.goroutine != self.goroutine2)

class Goroutine_EqualSameStackFrames(EqualityBaseTestCase):
    def runTest(self):
        warnings = populateGoroutine(self.goroutine,
                                     'goroutine 15 [select, 199 minutes]:',
                                     'sync.(*WaitGroup).Wait(0xc208110c00)',
	                             '   /usr/local/go/src/sync/waitgroup.go:132 +0x169',
                                     'github.com/control-center/go-zookeeper/zk.(*Conn).loop(0xc208129c70)',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:231 +0x76d',
                                     'github.com/control-center/go-zookeeper/zk.func·001()',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:149 +0x2c',
                                     'created by github.com/control-center/go-zookeeper/zk.ConnectWithDialer',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:153 +0x4b8')
        self.assertEqual(len(warnings), 0)

        warnings = populateGoroutine(self.goroutine2,
                                     'goroutine 24829 [IO wait]:',
                                     'sync.(*WaitGroup).Wait(0xc208110c00)',
	                             '   /usr/local/go/src/sync/waitgroup.go:132 +0x169',
                                     'github.com/control-center/go-zookeeper/zk.(*Conn).loop(0xc208129c70)',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:231 +0x76d',
                                     'github.com/control-center/go-zookeeper/zk.func·001()',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:149 +0x2c',
                                     'created by github.com/control-center/go-zookeeper/zk.ConnectWithDialer',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:153 +0x4b8')
        self.assertEqual(len(warnings), 0)

        self.assertTrue(self.goroutine == self.goroutine2)
        self.assertFalse(self.goroutine != self.goroutine2)

class Goroutine_NotEqualDifferentStackFrames(EqualityBaseTestCase):
    def runTest(self):
        warnings = populateGoroutine(self.goroutine,
                                     'goroutine 15 [select, 199 minutes]:',
                                     'sync.(*WaitGroup).Wait(0xc208110c00)',
	                             '   /usr/local/go/src/sync/waitgroup.go:132 +0x169',
                                     'github.com/control-center/go-zookeeper/zk.(*Conn).loop(0xc208129c70)',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:231 +0x76d',
                                     'github.com/control-center/go-zookeeper/zk.func·001()',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:149 +0x2c',
                                     'created by github.com/control-center/go-zookeeper/zk.ConnectWithDialer',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:153 +0x4b8')
        self.assertEqual(len(warnings), 0)

        warnings = populateGoroutine(self.goroutine2,
                                     'goroutine 24829 [IO wait]:',
                                     'sync.(*WaitGroup).Wait(0xc208110c00)',
	                             '   /usr/local/go/src/sync/waitgroup.go:132 +0x169',
                                     'github.com/control-center/go-zookeeper/zk.(*Conn).loop(0xc208129c70)',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:231 +0x76d',
                                     'github.com/control-center/go-zookeeper/zk.func·001()',
                                     # Difference is line number below:
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:555 +0x2c',
                                     'created by github.com/control-center/go-zookeeper/zk.ConnectWithDialer',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:153 +0x4b8')
        self.assertEqual(len(warnings), 0)

        self.assertFalse(self.goroutine == self.goroutine2)
        self.assertTrue(self.goroutine != self.goroutine2)


class Goroutine_NotEqualDifferentNumStackFrames(EqualityBaseTestCase):
    def runTest(self):
        warnings = populateGoroutine(self.goroutine,
                                     'goroutine 15 [select, 199 minutes]:',
                                     'sync.(*WaitGroup).Wait(0xc208110c00)',
	                             '   /usr/local/go/src/sync/waitgroup.go:132 +0x169',
                                     'github.com/control-center/go-zookeeper/zk.(*Conn).loop(0xc208129c70)',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:231 +0x76d',
                                     'github.com/control-center/go-zookeeper/zk.func·001()',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:149 +0x2c',
                                     'created by github.com/control-center/go-zookeeper/zk.ConnectWithDialer',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:153 +0x4b8')
        self.assertEqual(len(warnings), 0)

        warnings = populateGoroutine(self.goroutine2,
                                     'goroutine 24829 [IO wait]:',
                                     'sync.(*WaitGroup).Wait(0xc208110c00)',
	                             '   /usr/local/go/src/sync/waitgroup.go:132 +0x169',
                                     'github.com/control-center/go-zookeeper/zk.(*Conn).loop(0xc208129c70)',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:231 +0x76d',
                                     'github.com/control-center/go-zookeeper/zk.func·001()',
	                             '   /home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/control-center/go-zookeeper/zk/conn.go:153 +0x4b8')
        self.assertEqual(len(warnings), 0)

        self.assertFalse(self.goroutine == self.goroutine2)
        self.assertTrue(self.goroutine != self.goroutine2)
