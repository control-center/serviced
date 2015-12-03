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

import csv
import os
import shutil
import unittest

from subprocess import Popen, PIPE
from utils import populateGoroutine, ourDir

import gostack


def populate_stacktrace(stacktrace, goroutine_defs):
    for goroutineDef in goroutine_defs:
        newGoroutine = gostack.Goroutine()
        populateGoroutine(newGoroutine, *goroutineDef)
        stacktrace.addGoroutine(newGoroutine)


class StackTrace_Coalesce(unittest.TestCase):
    def setUp(self):
        self.stacktrace = gostack.StackTrace()
        populate_stacktrace(self.stacktrace, goroutineDefs)

    def runTest(self):
        coalesced = self.stacktrace.coalesce()
        for i in range(num_unique_goroutines):
            uniqueGoroutine = gostack.Goroutine()
            populateGoroutine(uniqueGoroutine, *goroutineDefs[i])
            uniqueKey = uniqueGoroutine.getKey()
            goroutines = coalesced[uniqueKey]
            self.assertEqual(len(goroutines), goroutineCounts[i])


class StackTrace_MonitorNoLabels(unittest.TestCase):
    def setUp(self):
        self.output_dir = './outdir'
        if os.path.exists(self.output_dir):
            shutil.rmtree(self.output_dir)

        self.stacktrace1 = gostack.StackTrace()
        populate_stacktrace(self.stacktrace1, goroutineDefs)
        self.stacktrace2 = gostack.StackTrace()
        populate_stacktrace(self.stacktrace2, goroutineDefs)
        populate_stacktrace(self.stacktrace2, goroutineDefs) # Twice as many goroutines
        self.stacktrace3 = gostack.StackTrace()
        populate_stacktrace(self.stacktrace3, goroutineDefs)

    def tearDown(self):
        if os.path.exists(self.output_dir):
            shutil.rmtree(self.output_dir)

    def runTest(self):
        monitored = gostack.MonitoredStackTrace(self.output_dir)
        monitored.addStackTrace('timeA', self.stacktrace1)
        monitored.addStackTrace('timeB', self.stacktrace2)
        monitored.addStackTrace('timeC', self.stacktrace3)
        monitored.save()

        with open(os.path.join(monitored.directory(), monitored.monitor_filename), 'r') as csvfile:
            try:
                datareader = csv.reader(csvfile, delimiter=',')
                i = 0
                for row in datareader:
                    if i == 0:
                        self.validate_header(row, False)
                    elif i == 1:
                        self.validate_small_value_row(i, row, 'timeA', None)
                    elif i == 2:
                        self.validate_large_value_row(i, row, 'timeB', None)
                    elif i == 3:
                        self.validate_small_value_row(i, row, 'timeC', None)
                    else:
                        self.fail('Extra row found in .csv file')
                    i += 1
            finally:
                csvfile.close()

        self.validate_legend_file(monitored)

    def validate_header(self, row, has_label):
        for i in range(len(row)):
            if i == 0:
                pass # Skip timestamp header--don't care what it is
            elif i == 1 and has_label:
                pass # Skip label header--don't care what it is
            else:
                if has_label:
                    goroutine_number = i - 1
                else:
                    goroutine_number = i
                self.assertEqual(row[i], 'Go' + str(goroutine_number), 'Header row, column {0} values not equal\n{1}'.format(i, row))

    def validate_small_value_row(self, row_num, row, timestamp, label):
        expected_values = [timestamp]
        if label is not None and len(label) > 0:
            expected_values.append(label)
        expected_values.extend(['6', '4', '2', '2', '2', '1'])
        self.validate_generic_row(row_num, row, expected_values)

    def validate_large_value_row(self, row_num, row, timestamp, label):
        expected_values = [timestamp]
        if label is not None and len(label) > 0:
            expected_values.append(label)
        expected_values.extend(['12', '8', '4', '4', '4', '2'])
        self.validate_generic_row(row_num, row, expected_values)

    def validate_generic_row(self, row_num, data_row, expected_row):
        self.assertEqual(len(data_row), len(expected_row),
                         'Row {0} has wrong number of elements: expected {1}, has {2}\n{3}'.format(row_num, len(expected_row), len(data_row), data_row))
        for i in range(len(data_row)):
            self.assertEqual(data_row[i], expected_row[i], 'Row {0}, column {1} values not equal\n{2}'.format(row_num, i, data_row))

    def validate_legend_file(self, monitored):
        actual_path = os.path.join(monitored.output_dir, monitored.legend_filename)
        expected_path = os.path.join(ourDir(), 'expectedlegend.txt')

        process = Popen(['diff', expected_path, actual_path], stdout=PIPE)
        process.communicate()
        exitCode = process.wait()
        self.assertEqual(exitCode, 0, 'Legend files do not match')


class StackTrace_MonitorWithLabels(StackTrace_MonitorNoLabels):
    def setUp(self):
        super(StackTrace_MonitorWithLabels, self).setUp()

    def tearDown(self):
        super(StackTrace_MonitorWithLabels, self).tearDown()

    def runTest(self):
        monitored = gostack.MonitoredStackTrace(self.output_dir)
        monitored.addStackTrace('timeA', self.stacktrace1, 'labelA')
        monitored.addStackTrace('timeB', self.stacktrace2, 'labelB')
        monitored.addStackTrace('timeC', self.stacktrace3, 'labelC')
        monitored.save()

        with open(os.path.join(monitored.directory(), 'gostack-monitor.csv'), 'r') as csvfile:
            try:
                datareader = csv.reader(csvfile, delimiter=',')
                i = 0
                for row in datareader:
                    if i == 0:
                        self.validate_header(row, True)
                    elif i == 1:
                        self.validate_small_value_row(i, row, 'timeA', 'labelA')
                    elif i == 2:
                        self.validate_large_value_row(i, row, 'timeB', 'labelB')
                    elif i == 3:
                        self.validate_small_value_row(i, row, 'timeC', 'labelC')
                    else:
                        self.fail('Extra row found in .csv file')
                    i += 1
            finally:
                csvfile.close()

        self.validate_legend_file(monitored)


# Counts of A-F defined in goroutineDefs below
goroutineCounts = [1, 4, 6, 2, 2, 2]

# Number of unique goroutines in goroueineDefs below
num_unique_goroutines = 6

goroutineDefs = [
    # --- Unique goroutines ----------------------
    # A
    [
        'goroutine 2365 [running]:',
        'github.com/fsouza/go-dockerclient.(*Client).do(0xc2081b2120, 0xd0ab10, 0x4, 0xc20921db00, 0x51, 0x0, 0x0, 0x0, 0x0, 0x0, ...)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/fsouza/go-dockerclient/client.go:373 +0x957',
        'github.com/fsouza/go-dockerclient.(*Client).WaitContainer(0xc2081b2120, 0xc209293a00, 0x40, 0x0, 0x0, 0x0)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/fsouza/go-dockerclient/container.go:817 +0xf1',
        'github.com/control-center/serviced/commons/docker.(*Client).WaitContainer(0xc208a7a650, 0xc209293a00, 0x40, 0x13ce028, 0x0, 0x0)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/commons/docker/client.go:179 +0x51',
        'github.com/control-center/serviced/commons/docker.func·003()',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/commons/docker/api.go:406 +0x65',
        'created by github.com/control-center/serviced/commons/docker.(*Container).Wait',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/commons/docker/api.go:408 +0x1ec',
    ],
    # B
    [
        'goroutine 14 [select, 199 minutes]:',
        'github.com/control-center/serviced/isvcs.(*IService).run(0xc208070900)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:462 +0x12a0',
        'created by github.com/control-center/serviced/isvcs.NewIService',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:180 +0x57f',
    ],
    # C
    [
        'goroutine 44 [IO wait, 199 minutes]:',
        'net.(*pollDesc).Wait(0xc2080a4ae0, 0x72, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:84 +0x47',
        'net.(*pollDesc).WaitRead(0xc2080a4ae0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:89 +0x43',
        'net.(*netFD).accept(0xc2080a4a80, 0x0, 0x7f906a6a13c8, 0xc2081795f0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_unix.go:419 +0x40b',
        'net.(*TCPListener).AcceptTCP(0xc2080fc320, 0x76c1d4, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/tcpsock_posix.go:234 +0x4e',
        'net/http.tcpKeepAliveListener.Accept(0xc2080fc320, 0x0, 0x0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1976 +0x4c',
        'net/http.(*Server).Serve(0xc2081ec060, 0x7f906a6a5fa0, 0xc2080fc320, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1728 +0x92',
        'net/http.(*Server).ListenAndServe(0xc2081ec060, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1718 +0x154',
        'net/http.ListenAndServe(0xd02270, 0x6, 0x7f906a6aa6c0, 0xc20817d0c0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1808 +0xba',
        'github.com/control-center/serviced/cli/api.func·014()',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:672 +0xaa',
        'created by github.com/control-center/serviced/cli/api.(*daemon).startAgent',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:673 +0xcd8',
    ],
    # D
    [
        'goroutine 38 [select]:',
        'github.com/control-center/serviced/proxy.(*TCPMux).loop(0xc20807cf40)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/proxy/mux.go:92 +0x643',
        'created by github.com/control-center/serviced/proxy.NewTCPMux',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/proxy/mux.go:47 +0x179',
    ],
    # E
    [
        'goroutine 487 [select, 195 minutes]:',
        'github.com/control-center/serviced/zzk/service.(*HostStateListener).Spawn(0xc208151000, 0xc2080c1680, 0xc208edd760, 0x19)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/zzk/service/host.go:223 +0x2d19',
        'github.com/control-center/serviced/zzk.func·008(0xc208edd760, 0x19)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/zzk/zzk.go:214 +0x114',
        'created by github.com/control-center/serviced/zzk.Listen',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/zzk/zzk.go:215 +0xfcb',
    ],
    # F
    [
        'goroutine 64 [select, 199 minutes]:',
        'github.com/control-center/serviced/utils.RunTTL(0x7f906a6ab210, 0x13ce028, 0xc208031380, 0xdf8475800, 0x4e94914f0000)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/utils/ttl.go:38 +0x355',
        'github.com/control-center/serviced/commons/docker.RunTTL(0xc208031380, 0xdf8475800, 0x4e94914f0000)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/commons/docker/ttl.go:28 +0x87',
        'github.com/control-center/serviced/node.func·010()',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/node/agent.go:788 +0xa4',
        'created by github.com/control-center/serviced/node.(*HostAgent).Start',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/node/agent.go:791 +0x2ba',
    ],
    # --- Now some repeats -------------------------
    # C
    [
        'goroutine 448 [IO wait, 199 minutes]:',
        'net.(*pollDesc).Wait(0xc2080a4ae0, 0x72, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:84 +0x47',
        'net.(*pollDesc).WaitRead(0xc2080a4ae0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:89 +0x43',
        'net.(*netFD).accept(0xc2080a4a80, 0x0, 0x7f906a6a13c8, 0xc2081795f0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_unix.go:419 +0x40b',
        'net.(*TCPListener).AcceptTCP(0xc2080fc320, 0x76c1d4, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/tcpsock_posix.go:234 +0x4e',
        'net/http.tcpKeepAliveListener.Accept(0xc2080fc320, 0x0, 0x0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1976 +0x4c',
        'net/http.(*Server).Serve(0xc2081ec060, 0x7f906a6a5fa0, 0xc2080fc320, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1728 +0x92',
        'net/http.(*Server).ListenAndServe(0xc2081ec060, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1718 +0x154',
        'net/http.ListenAndServe(0xd02270, 0x6, 0x7f906a6aa6c0, 0xc20817d0c0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1808 +0xba',
        'github.com/control-center/serviced/cli/api.func·014()',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:672 +0xaa',
        'created by github.com/control-center/serviced/cli/api.(*daemon).startAgent',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:673 +0xcd8',
    ],
    # B
    [
        'goroutine 104 [select, 199 minutes]:',
        'github.com/control-center/serviced/isvcs.(*IService).run(0xc208070900)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:462 +0x12a0',
        'created by github.com/control-center/serviced/isvcs.NewIService',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:180 +0x57f',
    ],
    # F
    [
        'goroutine 69 [select, 199 minutes]:',
        'github.com/control-center/serviced/utils.RunTTL(0x7f906a6ab210, 0x13ce028, 0xc208031380, 0xdf8475800, 0x4e94914f0000)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/utils/ttl.go:38 +0x355',
        'github.com/control-center/serviced/commons/docker.RunTTL(0xc208031380, 0xdf8475800, 0x4e94914f0000)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/commons/docker/ttl.go:28 +0x87',
        'github.com/control-center/serviced/node.func·010()',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/node/agent.go:788 +0xa4',
        'created by github.com/control-center/serviced/node.(*HostAgent).Start',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/node/agent.go:791 +0x2ba',
    ],
    # D
    [
        'goroutine 32 [select]:',
        'github.com/control-center/serviced/proxy.(*TCPMux).loop(0xc20807cf40)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/proxy/mux.go:92 +0x643',
        'created by github.com/control-center/serviced/proxy.NewTCPMux',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/proxy/mux.go:47 +0x179',
    ],
    # E
    [
        'goroutine 488 [select, 84 minutes]:',
        'github.com/control-center/serviced/zzk/service.(*HostStateListener).Spawn(0xc208151000, 0xc2080c1680, 0xc208edd760, 0x19)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/zzk/service/host.go:223 +0x2d19',
        'github.com/control-center/serviced/zzk.func·008(0xc208edd760, 0x19)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/zzk/zzk.go:214 +0x114',
        'created by github.com/control-center/serviced/zzk.Listen',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/zzk/zzk.go:215 +0xfcb',
    ],
    # C
    [
        'goroutine 129 [chan send]:',
        'net.(*pollDesc).Wait(0xc2080a4ae0, 0x72, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:84 +0x47',
        'net.(*pollDesc).WaitRead(0xc2080a4ae0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:89 +0x43',
        'net.(*netFD).accept(0xc2080a4a80, 0x0, 0x7f906a6a13c8, 0xc2081795f0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_unix.go:419 +0x40b',
        'net.(*TCPListener).AcceptTCP(0xc2080fc320, 0x76c1d4, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/tcpsock_posix.go:234 +0x4e',
        'net/http.tcpKeepAliveListener.Accept(0xc2080fc320, 0x0, 0x0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1976 +0x4c',
        'net/http.(*Server).Serve(0xc2081ec060, 0x7f906a6a5fa0, 0xc2080fc320, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1728 +0x92',
        'net/http.(*Server).ListenAndServe(0xc2081ec060, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1718 +0x154',
        'net/http.ListenAndServe(0xd02270, 0x6, 0x7f906a6aa6c0, 0xc20817d0c0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1808 +0xba',
        'github.com/control-center/serviced/cli/api.func·014()',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:672 +0xaa',
        'created by github.com/control-center/serviced/cli/api.(*daemon).startAgent',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:673 +0xcd8',
    ],
    # C
    [
        'goroutine 454 [IO wait, 199 minutes]:',
        'net.(*pollDesc).Wait(0xc2080a4ae0, 0x72, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:84 +0x47',
        'net.(*pollDesc).WaitRead(0xc2080a4ae0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:89 +0x43',
        'net.(*netFD).accept(0xc2080a4a80, 0x0, 0x7f906a6a13c8, 0xc2081795f0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_unix.go:419 +0x40b',
        'net.(*TCPListener).AcceptTCP(0xc2080fc320, 0x76c1d4, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/tcpsock_posix.go:234 +0x4e',
        'net/http.tcpKeepAliveListener.Accept(0xc2080fc320, 0x0, 0x0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1976 +0x4c',
        'net/http.(*Server).Serve(0xc2081ec060, 0x7f906a6a5fa0, 0xc2080fc320, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1728 +0x92',
        'net/http.(*Server).ListenAndServe(0xc2081ec060, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1718 +0x154',
        'net/http.ListenAndServe(0xd02270, 0x6, 0x7f906a6aa6c0, 0xc20817d0c0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1808 +0xba',
        'github.com/control-center/serviced/cli/api.func·014()',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:672 +0xaa',
        'created by github.com/control-center/serviced/cli/api.(*daemon).startAgent',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:673 +0xcd8',
    ],
    # B
    [
        'goroutine 23 [chan receive, 199 minutes]:',
        'github.com/control-center/serviced/isvcs.(*IService).run(0xc208070900)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:462 +0x12a0',
        'created by github.com/control-center/serviced/isvcs.NewIService',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:180 +0x57f',
    ],
    # B
    [
        'goroutine 78 [select, 199 minutes]:',
        'github.com/control-center/serviced/isvcs.(*IService).run(0xc208070900)',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:462 +0x12a0',
        'created by github.com/control-center/serviced/isvcs.NewIService',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:180 +0x57f',
    ],
    # C
    [
        'goroutine 887 [chan receive, 120 minutes]:',
        'net.(*pollDesc).Wait(0xc2080a4ae0, 0x72, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:84 +0x47',
        'net.(*pollDesc).WaitRead(0xc2080a4ae0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:89 +0x43',
        'net.(*netFD).accept(0xc2080a4a80, 0x0, 0x7f906a6a13c8, 0xc2081795f0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_unix.go:419 +0x40b',
        'net.(*TCPListener).AcceptTCP(0xc2080fc320, 0x76c1d4, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/tcpsock_posix.go:234 +0x4e',
        'net/http.tcpKeepAliveListener.Accept(0xc2080fc320, 0x0, 0x0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1976 +0x4c',
        'net/http.(*Server).Serve(0xc2081ec060, 0x7f906a6a5fa0, 0xc2080fc320, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1728 +0x92',
        'net/http.(*Server).ListenAndServe(0xc2081ec060, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1718 +0x154',
        'net/http.ListenAndServe(0xd02270, 0x6, 0x7f906a6aa6c0, 0xc20817d0c0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1808 +0xba',
        'github.com/control-center/serviced/cli/api.func·014()',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:672 +0xaa',
        'created by github.com/control-center/serviced/cli/api.(*daemon).startAgent',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:673 +0xcd8',
    ],
    # C
    [
        'goroutine 90 [IO wait, 99 minutes]:',
        'net.(*pollDesc).Wait(0xc2080a4ae0, 0x72, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:84 +0x47',
        'net.(*pollDesc).WaitRead(0xc2080a4ae0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:89 +0x43',
        'net.(*netFD).accept(0xc2080a4a80, 0x0, 0x7f906a6a13c8, 0xc2081795f0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/fd_unix.go:419 +0x40b',
        'net.(*TCPListener).AcceptTCP(0xc2080fc320, 0x76c1d4, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/tcpsock_posix.go:234 +0x4e',
        'net/http.tcpKeepAliveListener.Accept(0xc2080fc320, 0x0, 0x0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1976 +0x4c',
        'net/http.(*Server).Serve(0xc2081ec060, 0x7f906a6a5fa0, 0xc2080fc320, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1728 +0x92',
        'net/http.(*Server).ListenAndServe(0xc2081ec060, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1718 +0x154',
        'net/http.ListenAndServe(0xd02270, 0x6, 0x7f906a6aa6c0, 0xc20817d0c0, 0x0, 0x0)',
        '/home/homer/.gvm/gos/go1.4.2/src/net/http/server.go:1808 +0xba',
        'github.com/control-center/serviced/cli/api.func·014()',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:672 +0xaa',
        'created by github.com/control-center/serviced/cli/api.(*daemon).startAgent',
        '/home/homer/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:673 +0xcd8',
    ]
]
