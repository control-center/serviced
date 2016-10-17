#!/bin/bash
#######################################################
#
# Control Center API Acceptance Test
#
# You must define the serviced login credentials by setting
# the environment variables APPLICATION_USERID
# and APPLICATION_PASSWORD before running this script.
#
# Any command line arguments passed to the this script will be
# passed through to acceptance/runAPIacceptance.sh
#
#######################################################

# Use a directory unique to this test to avoid collisions with other kinds of tests
TEST_VAR_PATH=/tmp/serviced-api-acceptance/var
. test_lib.sh

trap cleanup EXIT
print_env_info

# Force a clean environment
echo "Starting Pre-test cleanup ..."
cleanup --ignore-errors
echo "Pre-test cleanup complete"

# Setup
install_prereqs
add_to_etc_hosts

start_serviced             && succeed "Serviced started within timeout"    || fail "serviced failed to start within $START_TIMEOUT seconds."

# build/start mock agents
cd ${DIR}
make mockAgent
cd ${DIR}/acceptance
sudo GOPATH=${GOPATH} PATH=${PATH} ./startMockAgents.sh --no-wait

# launch cucumber/capybara with colorized output disabled for better readability in Jenkins
CUCUMBER_OPTS=--no-color ./runAPIacceptance.sh -a https://${HOSTNAME} $*

# "trap cleanup EXIT", above, will handle cleanup
