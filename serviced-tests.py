import sys
import tempfile
import os
import subprocess
import argparse
import logging
from contextlib import contextmanager


log = logging.getLogger("serviced-tests")


SERVICED_ROOT = os.path.dirname(os.path.abspath(__file__))
DEVNULL = open(os.devnull, 'w')


def fail(msg):
    log.critical(msg)
    sys.exit(1)


@contextmanager
def elastic_server(port):
    log.info("Starting elastic on port %d " % port)
    yield
    log.info("Stopping elastic")


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

    coverage = parser.add_argument_group("Coverage Options")
    coverage.add_argument("--cover-html", required=False, help="output file for HTML coverage report")
    coverage.add_argument("--cover-xml", required=False, help="output file for Coberatura coverage report")

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


def get_tools():
    """
    Make sure you have go tools and stuff
    """
    pass

def get_gopath():
    gopath = os.environ.get("GOPATH")
    log.debug("Original GOPATH=%s" % (gopath or ""))
    try:
        godep_path = subprocess.check_output(["godep", "path"], cwd=SERVICED_ROOT)
    except subprocess.CalledProcessError:
        fail("Unable to run 'godep path'. Please check your environment.")
    gopath = "%s:%s" % (godep_path.strip(), gopath.strip())
    log.debug("Modified GOPATH=%s" % gopath)
    return gopath


def main(options):
    logging.basicConfig(level=logging.DEBUG if options.verbose else logging.INFO)

    if not any((options.unit, options.integration)):
        fail("No tests were specified to run. Please pass at least one of --unit or --integration.")

    log.debug("Running tests under serviced in %s" % SERVICED_ROOT)

    env = os.environ
    env.update({
        "GOPATH": get_gopath(),
        "SERVICED_HOME": SERVICED_ROOT,
    })

    tags = build_tags(options)

    runner = "gocov" if options.cover else "go"

    cmd = [runner, "test", "-tags", " ".join(tags)]
    cmd.extend(options.arguments)
    cmd.extend(options.packages or ["./..."])

    args = {}

    if options.cover:
        stdout = tempfile.NamedTemporaryFile()
        args["stdout"] = stdout

    log.debug("Running command: %s" % cmd)
    log.debug("Running in directory: %s" % SERVICED_ROOT)

    subprocess.call(
        cmd,
        env=env,
        cwd=SERVICED_ROOT,
        **args
    )








if __name__ == "__main__":
    options = args()
    main(options)