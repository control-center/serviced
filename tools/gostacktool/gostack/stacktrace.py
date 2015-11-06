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

# Maintains a list of the Goroutines.
class StackTrace(object):
    def __init__(self):
        self.goroutines = []

    def addGoroutine(self, goroutine):
        self.goroutines.append(goroutine)

    # Returns a CoalescedStackTrace
    def coalesce(self):
        coalesced = CoalescedStackTrace()
        for goroutine in self.goroutines:
            coalesced.addGoroutine(goroutine)
        return coalesced

    def __iter__(self):
        return iter(self.goroutines)

    def __repr__(self):
        return '<StackTrace: {0} goroutines>'.format(len(self.goroutines))

    def __str__(self):
        string = ""
        for goroutine in self.goroutines:
            string += str(goroutine) + '\n'
        return string

# A dictionary of the Goroutines, grouped per goroutine stack.
#   Key = Goroutine.getKey() (not really a meaningful value for human consumption).
#   Value = List of Goroutines with that same stack.
class CoalescedStackTrace(object):
    def __init__(self):
        self.goroutines = {}

    def addGoroutine(self, newGoroutine):
        for key, goroutineList in self.goroutines.iteritems():
            if newGoroutine == goroutineList[0]:
                goroutineList.append(newGoroutine)
                return
        self.goroutines[newGoroutine.getKey()] = [newGoroutine]

    def __iter__(self):
        return iter(sorted(self.goroutines.values(), key=lambda gor: len(gor), reverse=True))

    def __getitem__(self, key):
        return self.goroutines[key]

    def __repr__(self):
        return '<StackTrace: {0} goroutine groups>'.format(len(self.goroutines))

    def __str__(self):
        string = ""
        for goroutineList in self:
            string += str(len(goroutineList)) + ' of these:\n'
            string += 'goroutine XX [XXXX]:\n'
            goroutineStrings = str(goroutineList[0]).splitlines(True)
            string += ''.join(goroutineStrings[1:])
            string += '\n'
        return string
