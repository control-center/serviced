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

class ServicedTest(object):

    def __init__(self, servicedPath):
        self._servicedPath = servicedPath

    def _get_list(self):
        jsonblob = subprocess.check_output((self._servicedPath, "service", "list", "-v"))
        return json.loads(jsonblob)

    def _is_running(self, dockerID):
        output = subprocess.check_output(("docker", "ps", "--no-trunc"))
        for line in output.splitlines():
            if dockerID in line:
                return True
        return False

    def started(self):
        l = None
        for x in xrange(0, 5):
            try:
                l = self._get_list()
                break
            except Exception as ex:
                print ex
                pass
            time.sleep(2)
        if l is None:
            print "could not verify if services were started"
            sys.exit(1)
        print "lookging at %d services" % (len(l))
        for svc in l:
            if len(svc["Startup"]) == 0:
                print "skipping %s" % (svc["ID"])
                continue
            print "looking for status of %s" % (svc["ID"])
            for x in xrange(0, 5):
                if self._is_running(svc["ID"]):
                    print "%s was running" % (svc["ID"])
                    break
                time.sleep(2)
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
