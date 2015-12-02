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

# This script helps extract and examine Go stack traces (debug=2 format).

import argparse
import datetime
import os
import re
import sys
import time

import gostack


def parse_args():
    argparser = argparse.ArgumentParser(description='Performs analysis on a Go stack trace (debug=2 format).')
    subparsers = argparser.add_subparsers(dest='ANALYSIS', help='Type of analysis to perform.')

    subparser_echo = subparsers.add_parser('echo', help='Print the stack trace to stdout.')
    subparser_echo.add_argument('SOURCE', help='The URL of a service that supports the Go pprof package.')

    subparser_count = subparsers.add_parser('count', help='Count the number of occurrences of each stack. Prints to stdout.')
    subparser_count.add_argument('SOURCE', help='Either the path to a file containing a Go stack trace (debug=2 format), ' +
                                 'or the URL of a service that supports the Go pprof package.')

    subparser_monitor = subparsers.add_parser('monitor', help='Collect stack counts over time.' +
                                              ' Default is manually triggered (when you press Enter).' +
                                              ' Data files are written to a directory.')
    subparser_monitor.add_argument('-t', '--time', help='Perform automated collections instead of manual ones,' +
                                   ' at the specified interval. Append \'s\', \'m\', or \'h\' to specify seconds,' +
                                   ' minutes, or hours (example: \'20m\'). Default is minutes.',
                                   dest='TIME', required=False, default='Manual')
    subparser_monitor.add_argument('-d', '--directory', help='Directory to write results to. Default is \'./gostack-data\'.',
                                   dest='DIR', required=False)
    subparser_monitor.add_argument('SOURCE', help='The URL of a service that supports the Go pprof package.')

    return argparser.parse_args()

def main():
    args = parse_args()

    source = args.SOURCE
    # If an http or https URL, extend it to the listener and options we need for the proper stack trace
    if source.startswith('http'):
        if source[len(source)-1] != '/':
            source += '/'
        source += 'debug/pprof/goroutine?debug=2'

    # Perform the analysis requested by the user
    if args.ANALYSIS == 'echo':
        do_echo(source)

    elif args.ANALYSIS == 'count':
        do_count(source)

    elif args.ANALYSIS == 'monitor':
        if args.TIME == 'Manual':
            do_monitor_manual(source, args.DIR)
        else:
            do_monitor_auto(source, args.TIME, args.DIR)

def get_seconds(interval):
    if not re.match('^[0-9]+[smh]?$', interval):
        raise ValueError('Invalid time interval: {0}'.format(interval))
    last_char = interval[len(interval) - 1]
    if last_char.isdigit():
        # No unit specifier, so default is minutes
        return int(interval) * 60
    elif last_char == 's':
        return int(interval[:len(interval) - 1])
    elif last_char == 'm':
        return int(interval[:len(interval) - 1]) * 60
    elif last_char == 'h':
        return int(interval[:len(interval) - 1]) * 60 * 60

def do_echo(source):
    goParser = gostack.Parser()
    goParser.parse(source)
    print str(goParser.stacktrace)

def do_count(source):
    goParser = gostack.Parser()
    goParser.parse(source)
    coalesced = goParser.stacktrace.coalesce()
    print (str(coalesced))

# interval is in seconds
def do_monitor_auto(source, interval, outdir):
    seconds = get_seconds(interval)

    if interval[len(interval) - 1].isdigit():
        interval += 'm'
    print 'Collecting stacks every {0}. Press Ctrl-C to exit the program.'.format(interval)

    monitored = gostack.MonitoredStackTrace(outdir)
    try:
        while True:
            do_monitor_once(source, monitored)
            time.sleep(seconds)
    except KeyboardInterrupt:
        print # So the user prompt shows up on the next line

def do_monitor_manual(source, outdir):
    print 'Each time you want to count goroutines, type a label for that collection point'
    print 'and press Enter. Press Ctrl-C to exit the program.'
    monitored = gostack.MonitoredStackTrace(outdir)
    try:
        while True:
            label = raw_input('\nNext collection: ')
            if len(label) == 0:
                label = ' ' # Need to distinguish between manual's empty label and auto's no label at all
            do_monitor_once(source, monitored, label)
    except KeyboardInterrupt:
        print # So the user prompt shows up on the next line

def do_monitor_once(source, all_stacks, label=''):
    timestamp = datetime.datetime.utcnow()
    sys.stderr.write('{0} Collecting...'.format(timestamp.strftime('%X')))
    newParser = gostack.Parser()
    newParser.parse(source)
    all_stacks.addStackTrace(timestamp.strftime('%x %X'), newParser.stacktrace, label)
    all_stacks.save()
    sys.stderr.write('done\n')


if __name__ == "__main__":
    main()
