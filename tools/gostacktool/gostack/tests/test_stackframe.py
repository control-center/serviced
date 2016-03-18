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

class BaseTestCase(unittest.TestCase):
    def setUp(self):
        self.stackframe = gostack.StackFrame()

class EqualityBaseTestCase(BaseTestCase):
    def setUp(self):
        super(EqualityBaseTestCase, self).setUp()
        self.stackframe2 = gostack.StackFrame()

#-------------------------------------------
# Function line
#
# Formats:
#  github.com/fsouza/go-dockerclient.func·008(0xc20808a7e0, 0xc2080b0210)
#  github.com/control-center/serviced/cli/api.(*daemon).run(0xc2080f0180, 0x0, 0x0)
#  created by os/signal.init·1

class StackFrame_Function_GoodWithoutObjectPointer(BaseTestCase):
    def runTest(self):
        warnings = self.stackframe.parseFunctionLine('github.com/fsouza/go-dockerclient.func·008(0xc20808a7e0, 0xc2080b0210)')
        self.assertEqual(self.stackframe.function, 'github.com/fsouza/go-dockerclient.func·008')
        self.assertFalse(self.stackframe.iscreatedby)
        expectedArgs = ['0xc20808a7e0', '0xc2080b0210']
        self.assertEqual(self.stackframe.args, expectedArgs)
        self.assertEqual(len(warnings), 0)

class StackFrame_Function_GoodWithObjectPointer(BaseTestCase):
    def runTest(self):
        warnings = self.stackframe.parseFunctionLine('github.com/control-center/serviced/cli/api.(*daemon).run(0xc2080f0180, 0x0, 0x0)')
        self.assertEqual(self.stackframe.function, 'github.com/control-center/serviced/cli/api.(*daemon).run')
        self.assertFalse(self.stackframe.iscreatedby)
        expectedArgs = ['0xc2080f0180', '0x0', '0x0']
        self.assertEqual(self.stackframe.args, expectedArgs)
        self.assertEqual(len(warnings), 0)

class StackFrame_Function_GoodEmptyArgList(BaseTestCase):
    def runTest(self):
        warnings = self.stackframe.parseFunctionLine('schedule()')
        self.assertEqual(self.stackframe.function, 'schedule')
        self.assertFalse(self.stackframe.iscreatedby)
        self.assertEqual(len(self.stackframe.args), 0)
        self.assertEqual(len(warnings), 0)

class StackFrame_Function_GoodGoFunctionNumber(BaseTestCase):
    def runTest(self):
        warnings = self.stackframe.parseFunctionLine('created by os/signal.init·1')
        self.assertEqual(self.stackframe.function, 'os/signal.init·1')
        self.assertTrue(self.stackframe.iscreatedby)
        self.assertEqual(len(self.stackframe.args), 0)
        self.assertEqual(len(warnings), 0)

class StackFrame_Function_BadNoArgListNoObjectPointer(BaseTestCase):
    def runTest(self):
        with self.assertRaises(gostack.ParseError):
            self.stackframe.parseFunctionLine('os/signal.loop')

class StackFrame_Function_BadNoArgListWithObjectPointer(BaseTestCase):
    def runTest(self):
        with self.assertRaises(gostack.ParseError):
            self.stackframe.parseFunctionLine('github.com/control-center/serviced/cli/api.(*daemon).run')

class StackFrame_Function_BadMultiLine(BaseTestCase):
    def runTest(self):
        with self.assertRaises(gostack.ParseError):
            # No end paren, as if split over multiple lines
            self.stackframe.parseFunctionLine('github.com/control-center/serviced/cli/api.(*daemon).run(0xc2080f0180, 0x0,')

class StackFrame_Function_WarnExtraFields(BaseTestCase):
    def runTest(self):
        with self.assertRaises(gostack.ParseError):
            self.stackframe.parseFunctionLine('github.com/control-center/serviced/cli/api.(*daemon).run(0xc2080f0180, 0x0, 0x0) half-done')

#-------------------------------------------
# File line
#
# Format:
#   /home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:306 +0xb13

class StackFrame_FileGood(BaseTestCase):
    def runTest(self):
        warnings = self.stackframe.parseFileLine('/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:306 +0xb13')
        self.assertEquals(self.stackframe.filename, '/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go')
        self.assertEquals(self.stackframe.linenum, 306)
        self.assertEquals(self.stackframe.offset, '+0xb13')
        self.assertEquals(len(warnings), 0)

class StackFrame_FileGoodNoOffset(BaseTestCase):
    def runTest(self):
        warnings = self.stackframe.parseFileLine('/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:306')
        self.assertEquals(self.stackframe.filename, '/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go')
        self.assertEquals(self.stackframe.linenum, 306)
        self.assertEquals(self.stackframe.offset, '')
        self.assertEquals(len(warnings), 0)

class StackFrame_File_BadJustFilename(BaseTestCase):
    def runTest(self):
        with self.assertRaises(gostack.ParseError):
            self.stackframe.parseFileLine('/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go')

class StackFrame_File_BadNoLineNumber(BaseTestCase):
    def runTest(self):
        with self.assertRaises(gostack.ParseError):
            self.stackframe.parseFileLine('/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go +0x612')

class StackFrame_File_BadNoFilename(BaseTestCase):
    def runTest(self):
        with self.assertRaises(gostack.ParseError):
            self.stackframe.parseFileLine(':306 +0xb13')

class StackFrame_File_WarnExtraFields(BaseTestCase):
    def runTest(self):
        warnings = self.stackframe.parseFileLine('/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:306 +0xb13 and more')
        self.assertEquals(self.stackframe.filename, '/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go')
        self.assertEquals(self.stackframe.linenum, 306)
        self.assertEquals(self.stackframe.offset, '+0xb13')
        self.assertEquals(len(warnings), 1)
        self.assertTrue(warnings[0].lower().find('extra fields'))

#-------------------------------------------
# Equality

class StackFrame_Equality_Equal1(EqualityBaseTestCase):
    def runTest(self):
        # Fully qualified file name
        # Function args are different, rest is the same
        warnings = self.stackframe.parseFunctionLine('net.(*pollDesc).Wait(0xc208138680, 0x72, 0x0, 0x0)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe.parseFileLine('	/usr/local/go/src/net/fd_poll_runtime.go:84 +0x47')
        self.assertEquals(len(warnings), 0)

        warnings = self.stackframe2.parseFunctionLine('net.(*pollDesc).Wait(0xc208167f00, 0x78, 0x0, 0x0)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe2.parseFileLine('	/usr/local/go/src/net/fd_poll_runtime.go:84 +0x47')
        self.assertEquals(len(warnings), 0)

        self.assertTrue(self.stackframe == self.stackframe2)
        self.assertFalse(self.stackframe != self.stackframe2)

class StackFrame_Equality_Equal2(EqualityBaseTestCase):
    def runTest(self):
        # Relative file name
        # Function args are the same
        warnings = self.stackframe.parseFunctionLine('github.com/control-center/serviced/web.(*ServiceConfig).syncAllVhosts(0xc2081b84d0, 0xc20805af60, 0x0, 0x0)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/web/cpserver.go:378 +0x33e')
        self.assertEquals(len(warnings), 0)

        warnings = self.stackframe2.parseFunctionLine('github.com/control-center/serviced/web.(*ServiceConfig).syncAllVhosts(0xc2081b84d0, 0xc20805af60, 0x0, 0x0)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe2.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/web/cpserver.go:378 +0x33e')
        self.assertEquals(len(warnings), 0)

        self.assertTrue(self.stackframe == self.stackframe2)
        self.assertFalse(self.stackframe != self.stackframe2)

class StackFrame_Equality_Equal3(EqualityBaseTestCase):
    def runTest(self):
        # "Created by" function
        warnings = self.stackframe.parseFunctionLine('created by github.com/control-center/serviced/isvcs.NewIService')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe.parseFileLine('/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:186 +0x59f')
        self.assertEquals(len(warnings), 0)

        warnings = self.stackframe2.parseFunctionLine('created by github.com/control-center/serviced/isvcs.NewIService')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe2.parseFileLine('/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:186 +0x59f')
        self.assertEquals(len(warnings), 0)

        self.assertTrue(self.stackframe == self.stackframe2)
        self.assertFalse(self.stackframe != self.stackframe2)

class StackFrame_Equality_Equal4(EqualityBaseTestCase):
    def runTest(self):
        # goroutine function number
        # Function args are different, rest is the same
        warnings = self.stackframe.parseFunctionLine('github.com/fsouza/go-dockerclient.func·008(0xc20808a7e0, 0xc2080b0210)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/fsouza/go-dockerclient/event.go:285 +0x1b7')
        self.assertEquals(len(warnings), 0)

        warnings = self.stackframe2.parseFunctionLine('github.com/fsouza/go-dockerclient.func·008(0xffffffff, 0xeeee)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe2.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/fsouza/go-dockerclient/event.go:285 +0x1b7')
        self.assertEquals(len(warnings), 0)

        self.assertTrue(self.stackframe == self.stackframe2)
        self.assertFalse(self.stackframe != self.stackframe2)

class StackFrame_Equality_NotEqualDifferentFunctionName(EqualityBaseTestCase):
    def runTest(self):
        warnings = self.stackframe.parseFunctionLine('github.com/control-center/serviced/isvcs.(*Manager).loop(0xc20813cc80)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/manager.go:268 +0x9f')
        self.assertEquals(len(warnings), 0)

        warnings = self.stackframe2.parseFunctionLine('github.com/control-center/serviced/isvcs.(*Manager).loopy(0xc20813cc80)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe2.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/manager.go:268 +0x9f')
        self.assertEquals(len(warnings), 0)

        self.assertTrue(self.stackframe != self.stackframe2)
        self.assertFalse(self.stackframe == self.stackframe2)

class StackFrame_Equality_NotEqualDifferentFilename(EqualityBaseTestCase):
    def runTest(self):
        warnings = self.stackframe.parseFunctionLine('github.com/control-center/serviced/isvcs.(*Manager).loop(0xc20813cc80)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/manager.go:268 +0x9f')
        self.assertEquals(len(warnings), 0)

        warnings = self.stackframe2.parseFunctionLine('github.com/control-center/serviced/isvcs.(*Manager).loop(0xc20813cc80)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe2.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/agent.go:268 +0x9f')
        self.assertEquals(len(warnings), 0)

        self.assertTrue(self.stackframe != self.stackframe2)
        self.assertFalse(self.stackframe == self.stackframe2)

class StackFrame_Equality_NotEqualDifferentLinenum(EqualityBaseTestCase):
    def runTest(self):
        warnings = self.stackframe.parseFunctionLine('github.com/control-center/serviced/isvcs.(*Manager).loop(0xc20813cc80)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/manager.go:268 +0x9f')
        self.assertEquals(len(warnings), 0)

        warnings = self.stackframe2.parseFunctionLine('github.com/control-center/serviced/isvcs.(*Manager).loop(0xc20813cc80)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe2.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/manager.go:368 +0x9f')
        self.assertEquals(len(warnings), 0)

        self.assertTrue(self.stackframe != self.stackframe2)
        self.assertFalse(self.stackframe == self.stackframe2)

class StackFrame_Equality_NotEqualDifferentOffset(EqualityBaseTestCase):
    def runTest(self):
        warnings = self.stackframe.parseFunctionLine('github.com/control-center/serviced/isvcs.(*Manager).loop(0xc20813cc80)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/manager.go:268 +0x9f')
        self.assertEquals(len(warnings), 0)

        warnings = self.stackframe2.parseFunctionLine('github.com/control-center/serviced/isvcs.(*Manager).loop(0xc20813cc80)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe2.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/manager.go:268 +0x99')
        self.assertEquals(len(warnings), 0)

        self.assertTrue(self.stackframe != self.stackframe2)
        self.assertFalse(self.stackframe == self.stackframe2)

class StackFrame_Equality_NotEqualToNonStackFrame(BaseTestCase):
    def runTest(self):
        warnings = self.stackframe.parseFunctionLine('github.com/control-center/serviced/isvcs.(*Manager).loop(0xc20813cc80)')
        self.assertEquals(len(warnings), 0)
        warnings = self.stackframe.parseFileLine('	/home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/manager.go:268 +0x9f')
        self.assertEquals(len(warnings), 0)

        self.assertTrue(self.stackframe != 'github.com/control-center/serviced/isvcs.(*Manager).loop(0xc20813cc80)')
        self.assertFalse(self.stackframe == 'github.com/control-center/serviced/isvcs.(*Manager).loop(0xc20813cc80)')
