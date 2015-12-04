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

import csv
import os
import threading


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

    # Iterates through the lists of goroutine lists
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


# Multiple data structures to monitor the number of Goroutines over time.
class MonitoredStackTrace(object):
    def __init__(self, directory):
        if directory is not None and len(directory) > 0:
            self.output_dir = directory
        else:
            self.output_dir = './gostack-data'
        self.timestamps = []       # List of timestamps
        self.labels = []           # List of (optional) labels
        self.goroutineCounts = {}  # Dictionary: Key=Goroutine.getKey(), Value=List of counts (one per timestamp)
        self.goroutineStacks = {}  # Dictionary: Key=Goroutine.getKey(), Value=Full goroutine stack
        self.lock = threading.RLock()
        self.human_key_counter = 0
        self.monitor_filename = 'gostack-monitor.csv'
        self.legend_filename = 'gostack-legend.txt'

        # Make sure output dir exists
        if not os.path.exists(self.output_dir):
            os.makedirs(self.output_dir)

    def directory(self):
        return self.output_dir

    def addStackTrace(self, timestamp, stacktrace, label=''):
        if isinstance(stacktrace, CoalescedStackTrace):
            self.__addCoalescedStackTrace(timestamp, stacktrace, label)
        elif isinstance(stacktrace, StackTrace):
            self.__addCoalescedStackTrace(timestamp, stacktrace.coalesce(), label)
        else:
            raise TypeError('Object is not a stack trace.')

    def __addCoalescedStackTrace(self, timestamp, stacktrace, label):
        self.lock.acquire()
        try:
            self.timestamps.append(timestamp)
            if label is not None and len(label) > 0:
                self.labels.append(label)
            for goroutineList in stacktrace:
                key = goroutineList[0].getKey()
                if key in self.goroutineCounts:
                    self.goroutineCounts[key].append(len(goroutineList))
                else:
                    newList = [0] * (len(self.timestamps) - 1)
                    newList.append(len(goroutineList))
                    self.goroutineCounts[key] = newList
                    self.goroutineStacks[key] = str(goroutineList[0])
            for countList in self.goroutineCounts.values():
                if len(countList) < len(self.timestamps):
                    countList.append(0)
        finally:
            self.lock.release()

    def __repr__(self):
        return '<MonitoredStackTrace: {0} collection points on {1} goroutines>'.format(len(self.timestamps), len(self.goroutineCounts))

    def save(self):
        # Format is: Rows are collections, columns are goroutines.
        # This format is best for easily graphing some of the data in LibreOffice, Excel, etc.

        # In case we are interrupted, write the new data to temporary files then move them
        # over top of the previous versions
        temp_monitor_path = os.path.join(self.output_dir, 'gostack-monitor-TEMP.csv')
        temp_legend_path = os.path.join(self.output_dir, 'gostack-legend-TEMP.txt')
        official_monitor_path = os.path.join(self.output_dir, self.monitor_filename)
        official_legend_path = os.path.join(self.output_dir, self.legend_filename)

        self.lock.acquire()
        try:
            self.human_key_counter = 0

            writes_succeeded = False
            monitor_file = open(temp_monitor_path, 'w')
            try:
                csvDataFile = csv.writer(monitor_file)
                legend_file = open(temp_legend_path, 'w')
                try:
                    # Write CSV file
                    csvDataFile.writerow(self.__get_header_row())
                    for i in range(len(self.timestamps)):
                        csvDataFile.writerow(self.__get_data_row(i))

                    # Write legend file
                    self.human_key_counter = 0
                    for goroutine_key in iter(sorted(self.goroutineCounts, key=lambda gkey: sum(self.goroutineCounts[gkey]), reverse=True)):
                        humanKey = self.__get_next_human_key()
                        goroutineLines = str(self.goroutineStacks[goroutine_key]).splitlines(False)
                        legend_file.write('goroutine {0} [XXXX]:\n'.format(humanKey))
                        for line in goroutineLines[1:]:
                            legend_file.write(line + '\n')
                        legend_file.write('\n')

                    writes_succeeded = True
                finally:
                    legend_file.close()
            finally:
                monitor_file.close()

            if writes_succeeded:
                os.rename(temp_monitor_path, official_monitor_path)
                os.rename(temp_legend_path, official_legend_path)
        finally:
            if os.path.exists(temp_monitor_path):
                os.remove(temp_monitor_path)
            if os.path.exists(temp_legend_path):
                os.remove(temp_legend_path)
            self.lock.release()

    def __get_header_row(self):
        self.human_key_counter = 0
        keys = ['Timestamps']
        if len(self.labels) > 0:
            keys.append('Labels')
        for i in range(len(self.goroutineCounts)):
            keys.append(self.__get_next_human_key())
        return keys

    def __get_data_row(self, index):
        # Optional label, followed by timestamp, followed by count per stack
        data = [self.timestamps[index]]
        if len(self.labels) > 0:
            data.append(self.labels[index])
        for goroutine_key in iter(sorted(self.goroutineCounts, key=lambda gkey: sum(self.goroutineCounts[gkey]), reverse=True)):
            try:
                counts = self.goroutineCounts[goroutine_key]
                data.append(counts[index])
            except IndexError:
                data.append('ERR: Index {0} > {1}'.format(index, len(counts)))
            except Exception as ex:
                data.append('ERR: {0}'.format(ex))
        return data

    def __get_next_human_key(self):
        self.human_key_counter += 1
        return 'Go{0}'.format(self.human_key_counter)
