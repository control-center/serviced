#!/usr/bin/env python

# Copyright 2014 The Serviced Authors.
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

import subprocess
import json
import argparse
import sys
import time


def Try(f, times, sleep=2):
    ex = None
    for x in xrange(0, times):
        try:
            return f()
        except Exception as ex:
            print ex
        time.sleep(sleep)
    return None

class ServicedTest(object):

    def __init__(self, servicedPath):
        self._servicedPath = servicedPath

    def _get_list(self):
        return json.loads(subprocess.check_output((self._servicedPath, "service", "list", "-v")))

    def _docker_ps(self, all=False):
        args = ["docker", "ps", "-q"]
	if all:
            args.append("-a")
        return subprocess.check_output(args).splitlines()

    def _docker_inspect(self, dockerID):
        return json.loads(subprocess.check_output(("docker", "inspect", dockerID)))[0]

    def _docker_each_running(self):
        return { dockerID: self._docker_inspect(dockerID) for dockerID in self._docker_ps() }

    def _serviceIsRunning(self, serviceID, running=None):
        if running is None:
            running = self._docker_each_running()
        for dockerID, container in running.iteritems():
            for part in container["Config"]["Cmd"]:
                if serviceID in part:
                    return True
        return False

    def started(self):
        l = Try(self._get_list, 5)
        if l is None:
            print "could not verify if services were started"
            sys.exit(1)
        print "looking at %d services" % (len(l))
        running = self._docker_each_running()
        for svc in l:
            if len(svc["Startup"]) == 0:
                print "skipping %s" % (svc["ID"])
                continue
            print "looking for status of %s" % (svc["ID"])
            if self._serviceIsRunning(svc["ID"], running):
                print "%s was running" % (svc["ID"])
            else:
                print "svc %s was not found to be running" % (svc["ID"])
                sys.exit(1)

if __name__=="__main__":
    parser = argparse.ArgumentParser(description="smoke tests for serviced")
    parser.add_argument("--path", help="serviced path", default="./serviced")
    parser.add_argument("action")
    args = parser.parse_args()

    print args.action
    t = ServicedTest(args.path)
    f = getattr(t, args.action, None)
    if f is None:
        print "action %s was not found" % (args.action)
        sys.exit(1)
    try:
        f()
    except Exception as ex:
        print ex
        sys.exit(1)
