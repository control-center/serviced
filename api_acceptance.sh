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
# passed through to acceptance/runAPIAcceptance.sh
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

# add the local host as a CC host so there will be available IP assignments.
${SERVICED} host add --register "${HOSTNAME}:4979" "default" --memory "100%" -k /dev/null
if [ $? -ne 0 ]; then
    echo "Failed to add CC host for api acceptance test, exiting";
    exit 1;
fi

# build/start mock agents
cd ${DIR}
make mockAgent
cd ${DIR}/acceptance
sudo GOPATH=${GOPATH} PATH=${PATH} ./startMockAgents.sh --no-wait

# launch cucumber/capybara with colorized output disabled for better readability in Jenkins
SERVICED_BINARY=${SERVICED_BINARY} CUCUMBER_OPTS=--no-color ./runAPIAcceptance.sh -a https://${HOSTNAME} $*

# "trap cleanup EXIT", above, will handle cleanup
