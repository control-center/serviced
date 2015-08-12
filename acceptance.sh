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
SERVICED=${DIR}/serviced
IP=$(/sbin/ifconfig docker0 | grep 'inet addr:' | cut -d: -f2 | awk {'print $1'})
HOSTNAME=$(hostname)

#
# Setup of env vars required to build mockAgent
echo ==== ENV INFO =====
gvm use go1.4.2
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
    echo "Starting serviced..."
    sudo GOPATH=${GOPATH} PATH=${PATH} SERVICED_NOREGISTRY="true" ${SERVICED} -master -agent server &

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
    echo "Stopping serviced and mockAgent"
    sudo pkill -9 serviced
    sudo pkill -9 mockAgent
    sudo pkill -9 startMockAgent

    echo "Removing all docker containers"
    docker ps -a -q | xargs --no-run-if-empty docker rm -fv

    sudo rm -rf /tmp/serviced-root/var
}
trap cleanup EXIT


# Force a clean environment
cleanup

# Setup
install_prereqs
add_to_etc_hosts

start_serviced             && succeed "Serviced became leader within timeout"    || fail "serviced failed to become the leader within 120 seconds."

# build/start mock agents
cd ${DIR}
make mockAgent
cd ${DIR}/acceptance
sudo GOPATH=${GOPATH} PATH=${PATH} ./startMockAgents.sh --no-wait

# launch cucumber/capybara
./runUIAcceptance.sh -a https://${HOSTNAME} $*

# "trap cleanup EXIT", above, will handle cleanup
