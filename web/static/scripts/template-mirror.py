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

def fatal(*args, **kwargs):
    LOG.fatal(*args, **kwargs)
    sys.exit(1)

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
    _docker_op("pull", imageID)

def docker_push(imageID):
    _docker_op("push", imageID)

def docker_tag(imageID, newImageID):
    _docker_op("tag", imageID, newImageID)

def _docker_op(op, *args):
    try:
        sargs = ["docker", op]
        sargs.extend(args)
        subprocess.check_call(sargs)
    except Exception as ex:
        fatal("could not docker %s %s: %s", op, " ".join(args), ex)

def ping_registry(registry):
    try:
        url = "http://%s/v1/_ping" % (registry)
        urllib2.urlopen(url).read()
    except Exception as ex:
        fatal("could not ping registry %s: %s", url, ex)

class Main(object):

    def __init__(self, input_template, output_template, mirror):
        self._input_template = input_template
        self._output_template = output_template
        self._mirror = mirror

    def _load_template(self):
        try:
            with open(self._input_template) as f:
                return json.load(f)
        except Exception as ex:
            LOG.fatal("could not load template: %s", ex)
            sys.exit(1)

    def _dump_template(self, template):
        try:
            with open(self._output_template, "w") as f:
                json.dump(template, f, sort_keys=True, indent=4, separators=(',', ': '))
        except Exception as ex:
            LOG.fatal("could not write new template to %s: %s", self._output_template, ex)
            sys.exit(1)

    def run(self):

        if os.path.isfile(self._output_template):
            LOG.fatal("destination template file should not exist: %s", self._output_template)
            sys.exit(1)

        print "Pinging destination registry: %s" % (self._mirror)
        ping_registry(self._mirror)

        print "Extracting imageIDs from template"
        imageIDs = set()
        template = self._load_template()
        for svc in template['Services']:
            imageIDs.update(get_images(svc))

        for imageID in imageIDs:
            print "Pulling image %s" % (imageID)
            docker_pull(imageID)

        newImageIDs = []
        for imageID in imageIDs:
            newImageID = self._mirror + "/" + imageID
            print "Retagging %s to %s" % (imageID, newImageID)
            docker_tag(imageID, newImageID)
            newImageIDs.append(newImageID)
            for svc in template['Services']:
                remap_image_id(svc, imageID, newImageID)

        for imageID in newImageIDs:
            docker_push(imageID)

        print "writing new template to %s" % (self._output_template)
        description = template["Description"].split("|")[0]
        template["Description"] = description.strip() + " | %s mirror" % self._mirror
        self._dump_template(template)

if __name__=="__main__":
    parser = argparse.ArgumentParser(description="a tool to mirror images referenced in serviced templates")
    parser.add_argument("input_template", help="the template to mirror")
    parser.add_argument("mirror", help="the docker mirror to use (eg somehost.example.com:5000")
    parser.add_argument("output_template", help="the destination to write the modified template")
    args = parser.parse_args()
    main = Main(args.input_template, args.output_template, args.mirror)
    main.run()

