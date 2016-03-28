#!/usr/bin/env python

import sys
import tempfile
import time
import os
import subprocess
import argparse
import logging
from contextlib import contextmanager

import uuid

log = logging.getLogger("serviced-tests")


SERVICED_ROOT = os.path.dirname(os.path.abspath(__file__))
DEVNULL = open(os.devnull, 'w')


def fail(msg):
    log.critical(msg)
    sys.exit(1)


def which(prog):
    return subprocess.check_output("which %s" % prog, shell=True).strip()


@contextmanager
def elastic_server(port):
    try:
        log.info("Starting elastic on port %d " % port)
        container_name = str(uuid.uuid4())
        # TODO: handle already started
        # TODO: Get image name from serviced binary or isvcs.go
        # TODO: Wait for start more betterly
        cmd = ["docker", "run", "-d", "--name", container_name,
               "-p", "%d:9200" % port, "zenoss/serviced-isvcs:v38",
               "/opt/elasticsearch-0.90.9/bin/elasticsearch", "-f",
               "-Des.cluster.name=%s" % container_name,
               "-Dmulticast.enabled=false",
               "-Ddiscovery.zen.ping.multicast.ping.enabled=false"]
        subprocess.call(cmd)
        time.sleep(10)
        yield
    finally:
        log.info("Stopping elastic")
        subprocess.call(["docker", "stop", container_name])


@contextmanager
def dummy(*args, **kwargs):
    yield


def ensure_tool(executable, importpath):
    try:
        return which(executable)
    except subprocess.CalledProcessError:
        log.info("Installing %s tool" % executable)
        subprocess.call(["go", "get", importpath])
        return which(executable)


def has_dm_deferred_remove():
    """
    Test whether libdevmapper.h is new enough to support deferred remove
    functionality by compiling a file to see if the function is defined.
    """
    cmd = """
command -v gcc && ! (
cat <<EOF | gcc -ldevmapper -xc -
#include <libdevmapper.h>
int main() { dm_task_deferred_remove(NULL); }
EOF
)
"""
    try:
        subprocess.check_call(cmd, shell=True, stdout=DEVNULL, stderr=subprocess.STDOUT)
        return False
    except subprocess.CalledProcessError:
        return True


def args():
    """
    --all (some subset that is useful for someone)
    --packages (maybe positional?)
    """
    parser = argparse.ArgumentParser("serviced-tests")

    parser.add_argument("-v", "--verbose", action="store_true", help="verbose logging")

    types = parser.add_argument_group("Test Type")
    types.add_argument("--unit", action="store_true", help="pass the 'unit' build tag")
    types.add_argument("--integration", action="store_true", help="pass the 'integration' build tag")

    options = parser.add_argument_group("Test Options")
    options.add_argument("--quick", action="store_true", help="don't run tests with the '!quick' build constraint")
    options.add_argument("--root", action="store_true", help="run the tests as the root user")
    options.add_argument("--race", action="store_true", help="run tests with race detection")
    options.add_argument("--cover", action="store_true", help="run tests with coverage")
    options.add_argument("--tag", action="append", help="optional extra build tag (may be specified multiple times)")
    options.add_argument("--include_vendor", action="store_true", dest="include_vendor", help="run tests against the vendor directory")

    coverage = parser.add_argument_group("Coverage Options")
    coverage.add_argument("--cover-html", required=False, help="output file for HTML coverage report")
    coverage.add_argument("--cover-xml", required=False, help="output file for Cobertura coverage report")

    fixtures = parser.add_argument_group("Fixture Options")
    fixtures.add_argument("--elastic", action="store_true", help="start an elastic server before the test run")
    fixtures.add_argument("--elastic-port", type=int, help="elastic server port", default=9202)

    parser.add_argument("--packages", nargs="*", help="serviced packages to test, relative to the serviced root (defaults to ./...)")
    parser.add_argument("arguments", nargs=argparse.REMAINDER, help="optional arguments to be passed through to the test runner")

    return parser.parse_args()


def build_tags(options):
    tags = options.tag or []

    # We always need the daemon tag
    tags.append("daemon")

    if not has_dm_deferred_remove():
        tags.append("libdm_no_deferred_remove")

    if options.unit:
        tags.append("unit")

    if options.integration:
        tags.append('integration')

    if options.quick:
        tags.append('quick')

    if options.root:
        tags.append('root')

    log.debug("Using build tags: %s" % tags)
    return tags


def main(options):
    logging.basicConfig(level=logging.DEBUG if options.verbose else logging.INFO)

    if not any((options.unit, options.integration)):
        fail("No tests were specified to run. Please pass at least one of --unit or --integration.")

    log.debug("Running tests under serviced in %s" % SERVICED_ROOT)

    env = os.environ

    env["SERVICED_HOME"] = SERVICED_ROOT

    # Unset EDITOR so CLI tests won't fail
    env.pop("EDITOR", None)

    tags = build_tags(options)

    args = {}

    if options.cover:
        if options.race:
            fail("--race and --cover are mutually exclusive.")
        runner = ensure_tool("gocov", "github.com/axw/gocov/gocov")
        log.debug("Using gocov executable %s" % runner)
        if options.cover_html:
            ensure_tool("gocov-html", "gopkg.in/matm/v1/gocov-html")
        if options.cover_xml:
            ensure_tool("gocov-xml", "github.com/AlekSi/gocov-xml")
        stdout = tempfile.NamedTemporaryFile()
        log.debug("Writing temporary coverage output to %s" % stdout.name)
        args["stdout"] = stdout

    else:
        runner = which("go")
        log.debug("Using go executable %s" % runner)

    # TODO: Get a sudo session set up with an interactive proc
    cmd = ["sudo", "-E", "PATH=%s" % env["PATH"]] if options.root else []

    cmd.extend([runner, "test", "-tags", " ".join(tags)])

    usep1 = False

    if options.integration:
        if options.cover:
            env["GOMAXPROCS"] = "1"
        else:
            usep1 = True

    if options.race:
        log.debug("Running with race detection")
        env["GORACE"] = "history_size=7 halt_on_error=1"
        cmd.append("-race")
        usep1 = True

    if usep1:
        cmd.extend(['-p', '1'])

    packages = options.packages
    if not packages:
        if options.include_vendor:
            packages = "./..."
        else:
            packages = subprocess.check_output("go list ./... | grep -v vendor", shell=True).splitlines()
    cmd.extend(packages)

    passthru = options.arguments
    if passthru and passthru[0] == "--":
        passthru = passthru[1:]
    cmd.extend(passthru)

    log.debug("Running command: %s" % cmd)
    log.debug("Running in directory: %s" % SERVICED_ROOT)

    fixture = elastic_server if options.elastic else dummy

    with fixture(options.elastic_port):
        try:
                subprocess.check_call(
                    cmd,
                    env=env,
                    cwd=SERVICED_ROOT,
                    **args
                )
        except (subprocess.CalledProcessError, KeyboardInterrupt):
            sys.exit(1)

    if options.cover_html:
        log.debug("Converting coverage to HTML")
        with open(options.cover_html, 'w') as output:
            subprocess.call(["gocov-html", stdout.name], stdout=output)
        log.info("HTML output written to %s" % options.cover_html)

    if options.cover_xml:
        log.debug("Converting coverage to Cobertura XML")
        with open(options.cover_xml, 'w') as output:
            proc = subprocess.Popen(["gocov-xml", stdout.name], stdout=output, stdin=subprocess.PIPE)
            stdout.seek(0)
            proc.communicate(stdout.read())
        log.info("Cobertura output written to %s" % options.cover_xml)


if __name__ == "__main__":
    options = args()
    main(options)
