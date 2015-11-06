#!/usr/bin/env python

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

# This script helps extract and examine Go stack traces.

import argparse

import gostack


def parse_args():
    argparser = argparse.ArgumentParser(description='Performs analysis on a go stack trace.')
    argparser.add_argument('ANALYSIS', choices=['count', 'echo'], help='The type of analysis to perform.')
    argparser.add_argument('FILE', help='Name of the file containing the stack trace.')
    return argparser.parse_args()

def main():
    args = parse_args()

    # Read in the stack trace file
    goParser = gostack.Parser()
    goParser.parse(args.FILE)
    stacktrace = goParser.stacktrace

    # Perform the analysis requested by the user
    if args.ANALYSIS == 'echo':
        print str(stacktrace)
    elif args.ANALYSIS == 'count':
        coalesced = stacktrace.coalesce()
        print (str(coalesced))


if __name__ == "__main__":
    main()
