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

import sys

from exceptions import ParseError
from utils import raiseWithModifiedMessage

class StackFrame:
    def __init__(self):
        self.function = ''
        self.iscreatedby = False
        self.args = []
        self.filename = ''
        self.linenum = 0
        self.offset = ''

    # Returns a list of warnings for the line.
    # Raises an exception on a hard error.
    def parseFunctionLine(self, line):
        fieldnum = 0
        warnings = []
        line = line.strip()

        try:
            createdPrefix = 'created by '
            if not line.startswith(createdPrefix):
                # Formats:
                #  github.com/fsouza/go-dockerclient.func·008(0xc20808a7e0, 0xc2080b0210)
                #  github.com/control-center/serviced/cli/api.(*daemon).run(0xc2080f0180, 0x0, 0x0)

                # Verify line ends with argument list. Kind of need to do this because object pointers
                # in full function name (like "(*daemon)" above) might otherwise look like args.
                rightParenIndex = line.rfind(')')
                leftParenIndex = line.rfind('(')
                if rightParenIndex != (len(line) - 1) or leftParenIndex == -1:
                    raise ParseError('Not a function line (no argument list found)!')

                fieldnum = 1 # Function name (including path components)
                self.function, args = line.rsplit('(', 1)

                fieldnum = 2 # Function arguments
                args = args.rstrip(')')
                if len(args) > 0:
                    for arg in args.split(','):
                        self.addArg(arg.strip())
            else:
                # Format:
                #  created by os/signal.init·1
                fieldnum = 1 # Function name (including path components)
                self.function = line[len(createdPrefix):]
                self.iscreatedby = True

            fieldnum = 0 # Done processing fields

            # No need to look for extra fields, given our initial check that line ends with "(args)"

            return warnings

        except Exception as exc:
            raiseWithModifiedMessage(sys.exc_info(), self.formatFunctionMessage(str(exc), fieldnum))

    # Returns a list of warnings for the line.
    # Raises an exception on a hard error.
    def parseFileLine(self, line):
        # Formats:
	#   /home/kwalker/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:306 +0xb13
        #   /usr/local/go/src/compress/flate/deflate.go:150
        fieldnum = 0
        warnings = []
        line = line.strip()

        try:
            colonIndex = line.rfind(':')
            if colonIndex == -1:
                raise ParseError('Not a file line (no colon found)!')
            
            fieldnum = 1 # File name
            if colonIndex == 0:
                raise ParseError('No filename found!')
            self.filename, line = line.rsplit(':', 1)

            fieldnum = 2 # Line number
            if line.find(' ') == -1:
                linenum = line
                line = ''
            else:
                linenum, line = line.split(' ', 1)
            if not linenum.isdigit():
                raise ParseError('Expected integer line number, got: {0}'.format(linenum))
            self.linenum = int(linenum)

            fieldnum = 3 # Offset
            if line.find(' ') == -1:
                self.offset = line
                line = ''
            else:
                self.offset, line = line.split(' ', 1)

            fieldnum = 0 # Done processing fields

            # Verify no extra fields found on line
            if line is not None and len(line) > 0:
                warnings.append(self.formatFileMessage('Extra fields found: ''{0}'''.format(line), fieldnum))

            return warnings

        except Exception as exc:
            raiseWithModifiedMessage(sys.exc_info(), self.formatFileMessage(str(exc), fieldnum))

    def addArg(self, arg):
        self.args.append(arg)

    def formatFunctionMessage(self, msg, fieldnum):
        return 'function line, field {0}: {1}'.format(fieldnum, msg)

    def formatFileMessage(self, msg, fieldnum):
        return 'file line, field {0}: {1}'.format(fieldnum, msg)

    def getFileLine(self):
        return self.filename + ':' + str(self.linenum) + ' ' + self.offset

    def __repr__(self):
        if len(self.function) > 0:
            return '<StackFrame: ' + self.function + '>'
        else:
            return '<StackFrame (empty)>'

    def __str__(self):
        if not self.iscreatedby:
            string = self.function + '(' + ', '.join(self.args) + ')'
        else:
            string = 'created by ' + self.function
        string += '\n   ' + self.getFileLine()
        return string

    def __eq__(self, other):
        # other must be a StackFrame whose function, filename, linenum, and offset match ours
        if not isinstance(other, StackFrame):
            return False
        return (self.function == other.function and
                self.iscreatedby == other.iscreatedby and
                self.filename == other.filename and
                self.linenum == other.linenum and
                self.offset == other.offset)

    def __ne__(self, other):
        if not isinstance(other, StackFrame):
            return True
        return not (self == other)
