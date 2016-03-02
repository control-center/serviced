#!/bin/bash
#######################################################
#
# Control Center Acceptance Test
#
# You must define the serviced login credentials by setting
# the environment variables APPLICATION_USERID
# and APPLICATION_PASSWORD before running this script.
#
# Any command line arguments passed to the this script will be
# passed through to acceptance/runUIAcceptance.sh
#
#######################################################

DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
SERVICED=$(which serviced)
if [ -z "${SERVICED}" ]; then
    echo "ERROR: Can not find a serviced binary"
    exit 1
fi

SERVICED_STORAGE=$(which serviced-storage)
if [ -z "${SERVICED_STORAGE}" ]; then
    echo "ERROR: Can not find a serviced-storage binary"
    exit 1
fi

# Use a directory unique to this test to avoid collisions with other kinds of tests
SERVICED_VARPATH=/tmp/serviced-acceptance/var
IP=$(ip addr show docker0 | grep -w inet | awk {'print $2'} | cut -d/ -f1)
HOSTNAME=$(hostname)

#
# Setup of env vars required to build mockAgent
echo ==== ENV INFO =====
gvm use go1.6
go version
docker version
echo ===================

export GOPATH=$WORKSPACE/gopath
export PATH=$GOPATH/bin:$PATH

succeed() {
    echo ===== SUCCESS =====
    echo $@
    echo ===================
}

fail() {
    echo ====== FAIL ======
    echo $@
    echo ==================
    exit 1
}

# install prereqs
install_prereqs() {
    local wget_image="zenoss/ubuntu:wget"
    if ! docker inspect "${wget_image}" >/dev/null; then
        docker pull "${wget_image}"
       if ! docker inspect "${wget_image}" >/dev/null; then
            fail "ERROR: docker image "${wget_image}" is not available - wget tests will fail"
       fi
    fi
}

# Add the vhost to /etc/hosts so we can resolve it for the test
add_to_etc_hosts() {
    if [ -z "$(grep -e "^${IP} websvc.${HOSTNAME}" /etc/hosts)" ]; then
        sudo /bin/bash -c "echo ${IP} websvc.${HOSTNAME} >> /etc/hosts"
    fi
}

start_serviced() {
    # Note that we have to set SERVICED_MASTER instead of using the -master command line arg
    #   all of to force the proper subdirectories to be created under SERVICED_VARPATH
    echo "Starting serviced..."
    mkdir -p ${SERVICED_VARPATH}
    sudo GOPATH=${GOPATH} PATH=${PATH} SERVICED_VARPATH=${SERVICED_VARPATH} SERVICED_MASTER=1 ${SERVICED} --allow-loop-back=true --agent server &

    echo "Waiting 180 seconds for serviced to become the leader..."
    retry 180 wget --no-check-certificate http://${HOSTNAME}:443 -O- &>/dev/null
    return $?
}

retry() {
    TIMEOUT=$1
    shift
    COMMAND="$@"
    DURATION=0
    until [ ${DURATION} -ge ${TIMEOUT} ]; do
        TRY_COUNTDOWN=$[${TIMEOUT} - ${DURATION}]
        ${COMMAND}; RESULT=$?; [ ${RESULT} = 0 ] && break
        DURATION=$[$DURATION+1]
        sleep 1
    done
    return ${RESULT}
}

cleanup() {
    # remove the service to free up the disk space allocated in the devicemapper pool
    echo "Removing testsvc (if any) ..."
    sudo ${SERVICED} service remove testsvc

    echo "Stopping serviced and mockAgent ..."
    sudo pkill -9 serviced
    sudo pkill -9 mockAgent
    sudo pkill -9 startMockAgent

    echo "Removing all docker containers (if any) ..."
    docker ps -a -q | xargs --no-run-if-empty docker rm -fv

    # Get a list of mounted volumes before 'set -e' because the grep exits with 1
    # in scenarios where nothing is mounted.
    MOUNTED_VOLUMES=`cat /proc/mounts | grep ${SERVICED_VARPATH}/volumes 2>/dev/null`

    # By default, exit on the first error
    if [ "$1" != "--ignore-errors" ]; then
        set -e
    fi

    # Unmount all of the devicemapper volumes so that the mount points can be deleted
    if [ ! -z "${MOUNTED_VOLUMES}" ]; then
        echo "Unmounting ${SERVICED_VARPATH}/volumes/* ..."
        sudo umount -f ${SERVICED_VARPATH}/volumes/* 2>/dev/null
    fi

    # Disable the DM device so that the space for the loopback device is really freed
    # when we remove SERVICED_VARPATH/volumes
    echo "Cleaning up serviced storage ..."
    sudo ${SERVICED_STORAGE} -v disable ${SERVICED_VARPATH}/volumes

    echo "Removing up ${SERVICED_VARPATH} ..."
    sudo rm -rf ${SERVICED_VARPATH}

}
trap cleanup EXIT

# Force a clean environment
echo "Starting Pre-test cleanup ..."
cleanup --ignore-errors
echo "Pre-test cleanup complete"

# Setup
install_prereqs
add_to_etc_hosts

start_serviced             && succeed "Serviced became leader within timeout"    || fail "serviced failed to become the leader within 120 seconds."

# build/start mock agents
cd ${DIR}
make mockAgent
cd ${DIR}/acceptance
sudo GOPATH=${GOPATH} PATH=${PATH} ./startMockAgents.sh --no-wait

# launch cucumber/capybara with colorized output disabled for better readability in Jenkins
CUCUMBER_OPTS=--no-color ./runUIAcceptance.sh -a https://${HOSTNAME} $*

# "trap cleanup EXIT", above, will handle cleanup
