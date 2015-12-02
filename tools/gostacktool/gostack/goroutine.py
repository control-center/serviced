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

import re
import sys

from exceptions import ParseError
from utils import raiseWithModifiedMessage


class Goroutine:
    def __init__(self):
        self.id = -1
        self.state = ''
        self.waittime = 0
        self.lockedtothread = False
        self.frames = []

    def getKey(self):
        # Concatenate the file lines of each stack frame
        key = ""
        for frame in self.frames:
            key += frame.getFileLine() + '\n'
        return key

    # Returns a list of warnings for the line.
    # Raises an exception on a hard error.
    def parseLine(self, line):
        # Possible formats:
        #   goroutine 0 [idle]:
        #   goroutine 1 [chan receive, 30 minutes]:
        #   goroutine 17 [syscall, 31 minutes, locked to thread]:
        #   goroutine 34 [syscall, locked to thread]:
        fieldnum = 0
        warnings = []
        line = line.strip()
        
        try:
            fieldnum = 1 # 'goroutine'
            goword, line = line.split(' ', 1)
            if goword.lower() != 'goroutine':
                raise ParseError('First word ({0}) is not \'goroutine\'!'.format(goword))

            fieldnum = 2 # goroutine ID
            goId, line = line.split(' ', 1)
            if not goId.isdigit():
                raise ParseError('Expected integer goroutine ID, got: {0}'.format(goId))
            self.id = int(goId)
                
            fieldnum = 3 # State, wait time, etc.
            # Pull the state fields (the stuff between [ and ]) out from the rest of the line
            leftBraceIndex = line.find('[')
            rightBraceIndex = line.find(']')
            if leftBraceIndex == -1 or rightBraceIndex == -1 or rightBraceIndex < (leftBraceIndex + 2):
                raise ParseError('State info not found (or is empty)!')
            if leftBraceIndex > 0:
                warnings.append(self.formatMessage('Extra fields found before state info: {0}'.format(line[0:leftBraceIndex]), fieldnum))
            stateFields = line[leftBraceIndex+1:rightBraceIndex].split(',')
            line = line[rightBraceIndex+1:]
            # Now process each field
            for i in range(len(stateFields)):
                field = stateFields[i].strip()
                if i == 0: # First field is always state
                    self.state = field
                elif field == 'locked to thread':
                    self.lockedtothread = True
                elif re.match('^[0-9]+ minutes$', field): # Wait time
                    waittime, minutes = field.split(' ', 1)
                    if not waittime.isdigit():
                        raise ParseError('Expected integer wait time, got: {0}'.format(waittime[0]))
                    self.waittime = int(waittime)
                else:
                    warnings.append(self.formatMessage('Unknown field found in state info: {0}'.format(field), fieldnum))
                fieldnum += 1

            fieldnum = 0 # Done processing fields

            # Verify no extra fields found on line
            if line is not None and line != ':':
                warnings.append(self.formatMessage('Extra fields found: ''{0}'''.format(line), fieldnum))

            return warnings

        except Exception as exc:
            raiseWithModifiedMessage(sys.exc_info(), self.formatMessage(str(exc), fieldnum))

    def addFrame(self, frame):
        self.frames.append(frame)

    def formatMessage(self, msg, fieldnum):
        return 'goroutine line, field {0}: {1}'.format(fieldnum, msg)

    def __repr__(self):
        if self.id == -1:
            return '<Goroutine (empty)>'
        else:
            return '<Goroutine {0}: [{1}] {2} stack frames>'.format(self.id, self.state, len(self.frames))

    def __str__(self):
        string = 'goroutine ' + str(self.id) + ' [' + self.state
        if self.waittime != 0:
            string += ', ' + str(self.waittime) + ' minutes'
        if self.lockedtothread:
            string += ', locked to thread'
        string += ']:\n'

        for frame in self.frames:
            string += str(frame) + '\n'

        return string

    def __hash__(self):
        return len(self.getKey())
        
    def __eq__(self, other):
        # other must be a Goroutine whose stackframes match ours
        if not isinstance(other, Goroutine):
            return False
        if len(self.frames) != len(other.frames):
            return False
        for i in range(len(self.frames)):
            if self.frames[i] != other.frames[i]:
                return False
        return True

    def __ne__(self, other):
        if not isinstance(other, Goroutine):
            return True
        return not (self == other)

    def __lt__(self, other):
        if not isinstance(other, Goroutine):
            return False
        return self.id < other.id

    def __le__(self, other):
        if not isinstance(other, Goroutine):
            return False
        return self.id <= other.id

    def __gt__(self, other):
        if not isinstance(other, Goroutine):
            return True
        return self.id > other.id

    def __ge__(self, other):
        if not isinstance(other, Goroutine):
            return True
        return self.id >= other.id

