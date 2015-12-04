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

import os
import sys
import tempfile
import unittest

from cStringIO import StringIO
from subprocess import Popen, PIPE
from utils import ourDir

import gostack


class BaseTestCase(unittest.TestCase):
    def setUp(self):
        self.parser = gostack.Parser()

class TempFileTestCase(BaseTestCase):
    def setUp(self):
        super(TempFileTestCase, self).setUp()
        self.tempFile = tempfile.NamedTemporaryFile(delete=False)

    def tearDown(self):
        if os.path.isfile(self.tempFile.name):
            os.unlink(self.tempFile.name)

class ParserGood_DumpFormat(BaseTestCase):
    def runTest(self):
        testFile = ourDir() + "/stackdump.txt"
        self.parser.parse(testFile)
        self.assertEqual(self.parser.errors, 0)
        self.assertEqual(self.parser.warnings, 0)

class ParserGood_PanicFormat(BaseTestCase):
    def runTest(self):
        testFile = ourDir() + "/stackpanic.txt"
        self.parser.parse(testFile)
        self.assertEqual(self.parser.errors, 0)
        self.assertEqual(self.parser.warnings, 0)

class ParserGood_String(TempFileTestCase):
    def runTest(self):
        testFile = ourDir() + "/stackpanic.txt"
        
        self.parser.parse(testFile)
        self.assertEqual(self.parser.errors, 0)
        self.assertEqual(self.parser.warnings, 0)

        self.tempFile.write(str(self.parser.stacktrace))
        self.tempFile.close()
        
        process = Popen(['diff', '-wB', testFile, self.tempFile.name], stdout=PIPE)
        process.communicate()
        exitCode = process.wait()
        self.assertEqual(exitCode, 0, 'Files did not match')

class Capturing(list):
    def __enter__(self):
        self._stdout = sys.stdout
        sys.stdout = self._stringio = StringIO()
        return self
    def __exit__(self, *args):
        self.extend(self._stringio.getvalue().splitlines())
        sys.stdout = self._stdout
