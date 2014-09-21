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

# This script will take a compiled serviced template and attempt to 
# mirror images referenced in that image to a new registry. It will
# also create a new template file with update ImageIDs that reference
# the new mirror

import argparse
import subprocess
import os.path
import sys
import json
import logging
import urllib2

logging.basicConfig()
LOG = logging.getLogger(__name__)

def get_images(service):
    imageIDs = set()
    if service["ImageID"]:
        imageIDs.add(service["ImageID"])
    for svc in service["Services"]:
        imageIDs.update(get_images(svc))
    return imageIDs

def remap_image_id(service, oldImageID, newImageID):
    if service["ImageID"] == oldImageID:
        service["ImageID"] = newImageID
    for svc in service["Services"]:
        remap_image_id(svc, oldImageID, newImageID)

def docker_pull(imageID):
    subprocess.check_call(["docker", "pull", imageID])

def docker_push(imageID):
    subprocess.check_call(["docker", "push", imageID])

def ping_registry(registry):
    urllib2.urlopen("http://%s/v1/_ping" % (registry)).read()

def docker_tag(imageID, newImageID):
    subprocess.check_call(["docker", "tag", imageID, newImageID])
   
class Main(object):
    def __init__(self, input_template, output_template, mirror):
        self._input_template = input_template
        self._output_template = output_template
        self._mirror = mirror

    def run(self):

        if os.path.isfile(self._output_template):
            LOG.fatal("destination template file should not exist: %s", self._output_template)
            sys.exit(1)

        print "Pinging destination registry: %s" % (self._mirror)
        try:
            ping_registry(self._mirror)
        except Exception as ex:
            LOG.fatal("could not ping destination registry %s: %s", self._mirror, ex)
            sys.exit(1)

        print "Extracting imageIDs from template"
        imageIDs = set()
        try:
            template = self._load_template()
            for svc in template['Services']:
               imageIDs.update(get_images(svc))
        except Exception as ex:
            LOG.fatal("could not process template: %s", ex)
            sys.exit(1)

        try:
            for imageID in imageIDs:
                print "Pulling image %s" % (imageID)
                docker_pull(imageID)
        except Exception as ex:
            LOG.fatal("could not pull image %s: %s", imageID, ex)
            sys.exit(1)

        try:
            newImageIDs = []
            for imageID in imageIDs:
                newImageID = self._mirror + "/" + imageID
                print "Retagging %s to %s" % (imageID, newImageID)
                docker_tag(imageID, newImageID)
                newImageIDs.append(newImageID)
                for svc in template['Services']:
                    remap_image_id(svc, imageID, newImageID)

        except Exception as ex:
            LOG.fatal("could not retag %s to %s: %s", imageID, newImageID, ex)
            sys.exit(1)

        try:
            for imageID in newImageIDs:
                docker_push(imageID)
        except Exception as ex:
            LOG.fatal("could not docker push %s: %s", imageID, ex)
            sys.exit(1)

        try:
            print "writing new template to %s" % (self._output_template)
            with open(self._output_template, "w") as f:
                json.dump(template, f, sort_keys=True, indent=4, separators=(',', ': '))
        except Exception as ex:
            LOG.fatal("could not write new template to %s: %s", self._output_template, ex)
            sys.exit(1)

    def _load_template(self):
        try:
            with open(self._input_template) as f:
                return json.load(f)
        except Exception as ex:
            LOG.fatal("could not load template: %s", ex)
            sys.exit(1)

if __name__=="__main__":
    parser = argparse.ArgumentParser(description="a tool to mirror images referenced in serviced templates")
    parser.add_argument("input_template", help="the template to mirror")
    parser.add_argument("mirror", help="the docker mirror to use (eg somehost.example.com:5000")
    parser.add_argument("output_template", help="the destination to write the modified template")
    args = parser.parse_args()
    main = Main(args.input_template, args.output_template, args.mirror)
    main.run()

