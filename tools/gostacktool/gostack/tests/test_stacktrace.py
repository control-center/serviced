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

class StackTrace_Coalesce(unittest.TestCase):
    def setUp(self):
        self.stacktrace = gostack.StackTrace()
        for goroutineDef in goroutineDefs:
            newGoroutine = gostack.Goroutine()
            populateGoroutine(newGoroutine, *goroutineDef)
            self.stacktrace.addGoroutine(newGoroutine)

    def runTest(self):
        coalesced = self.stacktrace.coalesce()
        for i in range(6):
            uniqueGoroutine = gostack.Goroutine()
            populateGoroutine(uniqueGoroutine, *goroutineDefs[i])
            uniqueKey = uniqueGoroutine.getKey()
            goroutines = coalesced[uniqueKey]
            self.assertEqual(len(goroutines), goroutineCounts[i])


# Counts of A-F defined in goroutineDefs
goroutineCounts = [1, 4, 6, 2, 2, 2]

goroutineDefs = [
    # --- Original goroutines ----------------------
    # A
    [
        'goroutine 2365 [running]:',
        'github.com/fsouza/go-dockerclient.(*Client).do(0xc2081b2120, 0xd0ab10, 0x4, 0xc20921db00, 0x51, 0x0, 0x0, 0x0, 0x0, 0x0, ...)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/fsouza/go-dockerclient/client.go:373 +0x957',
        'github.com/fsouza/go-dockerclient.(*Client).WaitContainer(0xc2081b2120, 0xc209293a00, 0x40, 0x0, 0x0, 0x0)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/Godeps/_workspace/src/github.com/fsouza/go-dockerclient/container.go:817 +0xf1',
        'github.com/control-center/serviced/commons/docker.(*Client).WaitContainer(0xc208a7a650, 0xc209293a00, 0x40, 0x13ce028, 0x0, 0x0)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/commons/docker/client.go:179 +0x51',
        'github.com/control-center/serviced/commons/docker.func·003()',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/commons/docker/api.go:406 +0x65',
        'created by github.com/control-center/serviced/commons/docker.(*Container).Wait',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/commons/docker/api.go:408 +0x1ec',
    ],
    # B
    [
        'goroutine 14 [select, 199 minutes]:',
        'github.com/control-center/serviced/isvcs.(*IService).run(0xc208070900)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:462 +0x12a0',
        'created by github.com/control-center/serviced/isvcs.NewIService',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:180 +0x57f',
    ],
    # C
    [
        'goroutine 44 [IO wait, 199 minutes]:',
        'net.(*pollDesc).Wait(0xc2080a4ae0, 0x72, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:84 +0x47',
        'net.(*pollDesc).WaitRead(0xc2080a4ae0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:89 +0x43',
        'net.(*netFD).accept(0xc2080a4a80, 0x0, 0x7f906a6a13c8, 0xc2081795f0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_unix.go:419 +0x40b',
        'net.(*TCPListener).AcceptTCP(0xc2080fc320, 0x76c1d4, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/tcpsock_posix.go:234 +0x4e',
        'net/http.tcpKeepAliveListener.Accept(0xc2080fc320, 0x0, 0x0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1976 +0x4c',
        'net/http.(*Server).Serve(0xc2081ec060, 0x7f906a6a5fa0, 0xc2080fc320, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1728 +0x92',
        'net/http.(*Server).ListenAndServe(0xc2081ec060, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1718 +0x154',
        'net/http.ListenAndServe(0xd02270, 0x6, 0x7f906a6aa6c0, 0xc20817d0c0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1808 +0xba',
        'github.com/control-center/serviced/cli/api.func·014()',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:672 +0xaa',
        'created by github.com/control-center/serviced/cli/api.(*daemon).startAgent',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:673 +0xcd8',
    ],
    # D
    [
        'goroutine 38 [select]:',
        'github.com/control-center/serviced/proxy.(*TCPMux).loop(0xc20807cf40)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/proxy/mux.go:92 +0x643',
        'created by github.com/control-center/serviced/proxy.NewTCPMux',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/proxy/mux.go:47 +0x179',
    ],
    # E
    [
        'goroutine 487 [select, 195 minutes]:',
        'github.com/control-center/serviced/zzk/service.(*HostStateListener).Spawn(0xc208151000, 0xc2080c1680, 0xc208edd760, 0x19)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/zzk/service/host.go:223 +0x2d19',
        'github.com/control-center/serviced/zzk.func·008(0xc208edd760, 0x19)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/zzk/zzk.go:214 +0x114',
        'created by github.com/control-center/serviced/zzk.Listen',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/zzk/zzk.go:215 +0xfcb',
    ],
    # F
    [
        'goroutine 64 [select, 199 minutes]:',
        'github.com/control-center/serviced/utils.RunTTL(0x7f906a6ab210, 0x13ce028, 0xc208031380, 0xdf8475800, 0x4e94914f0000)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/utils/ttl.go:38 +0x355',
        'github.com/control-center/serviced/commons/docker.RunTTL(0xc208031380, 0xdf8475800, 0x4e94914f0000)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/commons/docker/ttl.go:28 +0x87',
        'github.com/control-center/serviced/node.func·010()',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/node/agent.go:788 +0xa4',
        'created by github.com/control-center/serviced/node.(*HostAgent).Start',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/node/agent.go:791 +0x2ba',
    ],
    # --- Now some repeats -------------------------
    # C
    [
        'goroutine 448 [IO wait, 199 minutes]:',
        'net.(*pollDesc).Wait(0xc2080a4ae0, 0x72, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:84 +0x47',
        'net.(*pollDesc).WaitRead(0xc2080a4ae0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:89 +0x43',
        'net.(*netFD).accept(0xc2080a4a80, 0x0, 0x7f906a6a13c8, 0xc2081795f0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_unix.go:419 +0x40b',
        'net.(*TCPListener).AcceptTCP(0xc2080fc320, 0x76c1d4, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/tcpsock_posix.go:234 +0x4e',
        'net/http.tcpKeepAliveListener.Accept(0xc2080fc320, 0x0, 0x0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1976 +0x4c',
        'net/http.(*Server).Serve(0xc2081ec060, 0x7f906a6a5fa0, 0xc2080fc320, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1728 +0x92',
        'net/http.(*Server).ListenAndServe(0xc2081ec060, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1718 +0x154',
        'net/http.ListenAndServe(0xd02270, 0x6, 0x7f906a6aa6c0, 0xc20817d0c0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1808 +0xba',
        'github.com/control-center/serviced/cli/api.func·014()',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:672 +0xaa',
        'created by github.com/control-center/serviced/cli/api.(*daemon).startAgent',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:673 +0xcd8',
    ],
    # B
    [
        'goroutine 104 [select, 199 minutes]:',
        'github.com/control-center/serviced/isvcs.(*IService).run(0xc208070900)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:462 +0x12a0',
        'created by github.com/control-center/serviced/isvcs.NewIService',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:180 +0x57f',
    ],
    # F
    [
        'goroutine 69 [select, 199 minutes]:',
        'github.com/control-center/serviced/utils.RunTTL(0x7f906a6ab210, 0x13ce028, 0xc208031380, 0xdf8475800, 0x4e94914f0000)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/utils/ttl.go:38 +0x355',
        'github.com/control-center/serviced/commons/docker.RunTTL(0xc208031380, 0xdf8475800, 0x4e94914f0000)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/commons/docker/ttl.go:28 +0x87',
        'github.com/control-center/serviced/node.func·010()',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/node/agent.go:788 +0xa4',
        'created by github.com/control-center/serviced/node.(*HostAgent).Start',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/node/agent.go:791 +0x2ba',
    ],
    # D
    [
        'goroutine 32 [select]:',
        'github.com/control-center/serviced/proxy.(*TCPMux).loop(0xc20807cf40)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/proxy/mux.go:92 +0x643',
        'created by github.com/control-center/serviced/proxy.NewTCPMux',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/proxy/mux.go:47 +0x179',
    ],
    # E
    [
        'goroutine 488 [select, 84 minutes]:',
        'github.com/control-center/serviced/zzk/service.(*HostStateListener).Spawn(0xc208151000, 0xc2080c1680, 0xc208edd760, 0x19)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/zzk/service/host.go:223 +0x2d19',
        'github.com/control-center/serviced/zzk.func·008(0xc208edd760, 0x19)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/zzk/zzk.go:214 +0x114',
        'created by github.com/control-center/serviced/zzk.Listen',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/zzk/zzk.go:215 +0xfcb',
    ],
    # C
    [
        'goroutine 129 [chan send]:',
        'net.(*pollDesc).Wait(0xc2080a4ae0, 0x72, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:84 +0x47',
        'net.(*pollDesc).WaitRead(0xc2080a4ae0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:89 +0x43',
        'net.(*netFD).accept(0xc2080a4a80, 0x0, 0x7f906a6a13c8, 0xc2081795f0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_unix.go:419 +0x40b',
        'net.(*TCPListener).AcceptTCP(0xc2080fc320, 0x76c1d4, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/tcpsock_posix.go:234 +0x4e',
        'net/http.tcpKeepAliveListener.Accept(0xc2080fc320, 0x0, 0x0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1976 +0x4c',
        'net/http.(*Server).Serve(0xc2081ec060, 0x7f906a6a5fa0, 0xc2080fc320, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1728 +0x92',
        'net/http.(*Server).ListenAndServe(0xc2081ec060, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1718 +0x154',
        'net/http.ListenAndServe(0xd02270, 0x6, 0x7f906a6aa6c0, 0xc20817d0c0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1808 +0xba',
        'github.com/control-center/serviced/cli/api.func·014()',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:672 +0xaa',
        'created by github.com/control-center/serviced/cli/api.(*daemon).startAgent',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:673 +0xcd8',
    ],
    # C
    [
        'goroutine 454 [IO wait, 199 minutes]:',
        'net.(*pollDesc).Wait(0xc2080a4ae0, 0x72, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:84 +0x47',
        'net.(*pollDesc).WaitRead(0xc2080a4ae0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:89 +0x43',
        'net.(*netFD).accept(0xc2080a4a80, 0x0, 0x7f906a6a13c8, 0xc2081795f0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_unix.go:419 +0x40b',
        'net.(*TCPListener).AcceptTCP(0xc2080fc320, 0x76c1d4, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/tcpsock_posix.go:234 +0x4e',
        'net/http.tcpKeepAliveListener.Accept(0xc2080fc320, 0x0, 0x0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1976 +0x4c',
        'net/http.(*Server).Serve(0xc2081ec060, 0x7f906a6a5fa0, 0xc2080fc320, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1728 +0x92',
        'net/http.(*Server).ListenAndServe(0xc2081ec060, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1718 +0x154',
        'net/http.ListenAndServe(0xd02270, 0x6, 0x7f906a6aa6c0, 0xc20817d0c0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1808 +0xba',
        'github.com/control-center/serviced/cli/api.func·014()',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:672 +0xaa',
        'created by github.com/control-center/serviced/cli/api.(*daemon).startAgent',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:673 +0xcd8',
    ],
    # B
    [
        'goroutine 23 [chan receive, 199 minutes]:',
        'github.com/control-center/serviced/isvcs.(*IService).run(0xc208070900)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:462 +0x12a0',
        'created by github.com/control-center/serviced/isvcs.NewIService',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:180 +0x57f',
    ],
    # B
    [
        'goroutine 78 [select, 199 minutes]:',
        'github.com/control-center/serviced/isvcs.(*IService).run(0xc208070900)',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:462 +0x12a0',
        'created by github.com/control-center/serviced/isvcs.NewIService',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/isvcs/container.go:180 +0x57f',
    ],
    # C
    [
        'goroutine 887 [chan receive, 120 minutes]:',
        'net.(*pollDesc).Wait(0xc2080a4ae0, 0x72, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:84 +0x47',
        'net.(*pollDesc).WaitRead(0xc2080a4ae0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:89 +0x43',
        'net.(*netFD).accept(0xc2080a4a80, 0x0, 0x7f906a6a13c8, 0xc2081795f0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_unix.go:419 +0x40b',
        'net.(*TCPListener).AcceptTCP(0xc2080fc320, 0x76c1d4, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/tcpsock_posix.go:234 +0x4e',
        'net/http.tcpKeepAliveListener.Accept(0xc2080fc320, 0x0, 0x0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1976 +0x4c',
        'net/http.(*Server).Serve(0xc2081ec060, 0x7f906a6a5fa0, 0xc2080fc320, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1728 +0x92',
        'net/http.(*Server).ListenAndServe(0xc2081ec060, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1718 +0x154',
        'net/http.ListenAndServe(0xd02270, 0x6, 0x7f906a6aa6c0, 0xc20817d0c0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1808 +0xba',
        'github.com/control-center/serviced/cli/api.func·014()',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:672 +0xaa',
        'created by github.com/control-center/serviced/cli/api.(*daemon).startAgent',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:673 +0xcd8',
    ],
    # C
    [
        'goroutine 90 [IO wait, 99 minutes]:',
        'net.(*pollDesc).Wait(0xc2080a4ae0, 0x72, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:84 +0x47',
        'net.(*pollDesc).WaitRead(0xc2080a4ae0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_poll_runtime.go:89 +0x43',
        'net.(*netFD).accept(0xc2080a4a80, 0x0, 0x7f906a6a13c8, 0xc2081795f0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/fd_unix.go:419 +0x40b',
        'net.(*TCPListener).AcceptTCP(0xc2080fc320, 0x76c1d4, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/tcpsock_posix.go:234 +0x4e',
        'net/http.tcpKeepAliveListener.Accept(0xc2080fc320, 0x0, 0x0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1976 +0x4c',
        'net/http.(*Server).Serve(0xc2081ec060, 0x7f906a6a5fa0, 0xc2080fc320, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1728 +0x92',
        'net/http.(*Server).ListenAndServe(0xc2081ec060, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1718 +0x154',
        'net/http.ListenAndServe(0xd02270, 0x6, 0x7f906a6aa6c0, 0xc20817d0c0, 0x0, 0x0)',
        '/home/smousa/.gvm/gos/go1.4.2/src/net/http/server.go:1808 +0xba',
        'github.com/control-center/serviced/cli/api.func·014()',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:672 +0xaa',
        'created by github.com/control-center/serviced/cli/api.(*daemon).startAgent',
        '/home/smousa/src/europa/src/golang/src/github.com/control-center/serviced/cli/api/daemon.go:673 +0xcd8',
    ]
]
