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

from __future__ import print_function

import sys
import urllib

from goroutine import Goroutine
from stackframe import StackFrame
from stacktrace import StackTrace


class State:
    findGoroutine = 0
    getFunction = 1
    getFile = 2
    fileEnd = 3


class Parser:
    def __init__(self):
        self.parsed = False
        self.filepath = None
        self.errors = 0
        self.warnings = 0

    def parse(self, url):
        self.linenum = 0
        self.stacktrace = StackTrace()
        self.errors = 0
        self.warnings = 0

        data = urllib.urlopen(url)

        try:
            self.state = State.findGoroutine

            for line in data:
                self.linenum += 1
                self.processLine(line.strip())

            if self.state != State.findGoroutine:
                self.finishGoroutine(True)
        finally:
            self.parsed = True

    def processLine(self, line):
        if self.state == State.findGoroutine:
            if line.startswith('goroutine'):
                self.processGoroutineLine(line)
            # Otherwise, still searching...

        elif self.state == State.getFunction:
            self.processFunctionLine(line)
            
        elif self.state == State.getFile:
            self.processFileLine(line)
            
        elif self.state == State.fileEnd:
            # Blank line indicates end of this goroutine
            if not line:
                self.finishGoroutine(True)
            # Non-blank line starting with 'goroutine' indicates start of next goroutine
            elif line.startswith('goroutine'):
                self.finishGoroutine(True)
                self.processGoroutineLine(line)
            # Other non-blank line should be another stack frame
            else:
                self.state == State.getFunction
                self.processFunctionLine(line)

    def processGoroutineLine(self, line):
        try:
            self.currentGoroutine = Goroutine()

            warnings = self.currentGoroutine.parseLine(line)
            self.reportWarnings(warnings, line)

            self.state = State.getFunction

        except Exception as exc:
            self.reportError(str(exc), line)
            # Skip to next goroutine
            self.finishGoroutine(False)

    def processFunctionLine(self, line):
        try:
            self.currentStackFrame = StackFrame()

            warnings = self.currentStackFrame.parseFunctionLine(line)
            self.reportWarnings(warnings, line)

            self.state = State.getFile

        except Exception as exc:
            self.reportError(str(exc), line)
            # Skip to next goroutine
            self.finishStackFrame(False)
            self.finishGoroutine(True)

    def processFileLine(self, line):
        if self.currentStackFrame is None:
            raise InternalError('No StackFrame instance available!')
        
        try:
            warnings = self.currentStackFrame.parseFileLine(line)
            self.reportWarnings(warnings, line)

            self.finishStackFrame(True) # Sets self.state

        except Exception as exc:
            self.reportError(str(exc), line)
            # Skip to next goroutine
            self.finishStackFrame(False)
            self.finishGoroutine(True)
            
    def finishGoroutine(self, keepIt):
        if keepIt and self.currentGoroutine is not None:
            self.stacktrace.addGoroutine(self.currentGoroutine)
        self.currentGoroutine = None
        self.state = State.findGoroutine
        
    def finishStackFrame(self, keepIt):
        if keepIt and self.currentStackFrame is not None:
            self.currentGoroutine.addFrame(self.currentStackFrame)
        self.currentStackFrame = None
        self.state = State.fileEnd

    def addLineNumber(self, msg):
        return 'Line {0}: {1}'.format(self.linenum, msg)

    def reportWarnings(self, warnings, line):
        if len(warnings) > 0:
            for warning in warnings:
                self.reportWarning(warning)
            print('         {0}'.format(self.addLineNumber(line)))

    def reportWarning(self, msg):
        self.warnings += 1
        print('WARNING: {0}'.format(self.addLineNumber(msg)), file=sys.stderr)

    def reportError(self, msg, line):
        self.errors += 1
        print('ERROR  : {0}'.format(self.addLineNumber(msg)), file=sys.stderr)
        print('       : {0}'.format(self.addLineNumber(line)))
