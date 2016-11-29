#!/bin/bash
#######################################################
#
# test_lib.sh - Common library of test-related methods
#
# This script is used by smoke.sh, acceptance.sh and
# any other bash-based test harness script which needs
# to perform common actions like setting up and tearing
# a Control Center storage.
#
#######################################################

SERVICED=$(PATH=${GOPATH}/bin:${PATH} which serviced)
if [ -z "${SERVICED}" ]; then
    echo "ERROR: Can not find a serviced binary"
    exit 1
fi
SERVICED_BINARY="${SERVICED}"
SERVICED_STORAGE=$(PATH=${GOPATH}/bin:${PATH} which serviced-storage)
if [ -z "${SERVICED_STORAGE}" ]; then
    echo "ERROR: Can not find a serviced-storage binary"
    exit 1
fi

export START_TIMEOUT=300
DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
IP=$(ip addr show docker0 | grep -w inet | awk {'print $2'} | cut -d/ -f1)
HOSTNAME=$(hostname)

export SERVICED_ETC_PATH=${TEST_VAR_PATH}/etc
export SERVICED_VOLUMES_PATH=${TEST_VAR_PATH}/volumes
export SERVICED_ISVCS_PATH=${TEST_VAR_PATH}/isvcs
export SERVICED_BACKUPS_PATH=${TEST_VAR_PATH}/backups
export TEST_CONFIG_FILE=${TEST_VAR_PATH}/serviced.default

# Using an empty default file insulates the smoke tests from any random configuration on the current machine,
# and, on the build slaves, it eliminates noise from the logs re: "could not read default configs"
setup_serviced_config() {
    mkdir -p ${TEST_VAR_PATH}
    rm -f ${TEST_CONFIG_FILE}
    touch ${TEST_CONFIG_FILE}

    mkdir -p ${SERVICED_ETC_PATH}
    mkdir -p ${SERVICED_VOLUMES_PATH}
    mkdir -p ${SERVICED_ISVCS_PATH}
    mkdir -p ${SERVICED_BACKUPS_PATH}

    echo "SERVICED_ETC_PATH=${SERVICED_ETC_PATH}"         >> ${TEST_CONFIG_FILE}
    echo "SERVICED_VOLUMES_PATH=${SERVICED_VOLUMES_PATH}" >> ${TEST_CONFIG_FILE}
    echo "SERVICED_ISVCS_PATH=${SERVICED_ISVCS_PATH}"     >> ${TEST_CONFIG_FILE}
    echo "SERVICED_BACKUPS_PATH=${SERVICED_BACKUPS_PATH}" >> ${TEST_CONFIG_FILE}
    echo "SERVICED_MASTER=1"                              >> ${TEST_CONFIG_FILE}
    SERVICED="${SERVICED} --config-file ${TEST_CONFIG_FILE}"

    cp ${DIR}/pkg/logconfig-cli.yaml    ${SERVICED_ETC_PATH}
    cp ${DIR}/pkg/logconfig-server.yaml ${SERVICED_ETC_PATH}

    echo "Contents of TEST_CONFIG_FILE:"
    cat ${TEST_CONFIG_FILE}
}

print_env_info() {
    echo ==== ENV INFO =====
    go version
    docker version
    echo "TEST_VAR_PATH=${TEST_VAR_PATH}"
    echo "TEST_CONFIG_FILE=${TEST_CONFIG_FILE}"
    echo ===================
}

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
            fail "ERROR: docker inspect "${wget_image}" failed - wget tests will fail"
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
    #   all of to force the proper subdirectories to be created under TEST_VAR_PATH
    echo "Starting serviced ..."

    setup_serviced_config

    sudo GOPATH=${GOPATH} PATH=${PATH} \
        ${SERVICED} ${SERVICED_OPTS} \
        --allow-loop-back=true server &

    echo "Waiting $START_TIMEOUT seconds for serviced to start ..."
    retry $START_TIMEOUT  wget --no-check-certificate http://${HOSTNAME}:443 -O- &>/dev/null
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
    echo "Starting test cleanup"
    # remove the service to free up the disk space allocated in the devicemapper pool
    echo "Removing testsvc (if any) ..."
    sudo ${SERVICED} service remove testsvc

    echo "Stopping serviced ..."
    sudo pkill -9 serviced

    echo "Stopping mockAgent (if any) ..."
    sudo pkill -9 mockAgent
    sudo pkill -9 startMockAgent

    echo "Removing all docker containers (if any) ..."
    docker ps -qa | xargs --no-run-if-empty docker rm -fv

    # Get a list of mounted volumes before 'set -e' because the grep exits with 1
    # in scenarios where nothing is mounted.
    MOUNTED_VOLUMES=`grep ${SERVICED_VOLUMES_PATH} /proc/mounts | cut -d' ' -f2`
    EXPORTED_VOLUMES=`grep /exports/serviced_volumes_v2 /proc/mounts | cut -d' ' -f2`

    # By default, exit on the first error
    if [ "$1" != "--ignore-errors" ]; then
        set -e
    fi

    # Unmount all of the devicemapper volumes so that the mount points can be deleted
    if [ ! -z "${MOUNTED_VOLUMES}" ]; then
        echo "Unmounting ${SERVICED_VOLUMES_PATH}/* ..."
        echo "MOUNTED_VOLUMES=${MOUNTED_VOLUMES}"
        sudo umount -f ${MOUNTED_VOLUMES} 2>/dev/null
    fi
    if [ ! -z "${EXPORTED_VOLUMES}" ]; then
        echo "Unmounting /exports/serviced_volumes_v2/* ..."
        echo "EXPORTED_VOLUMES=${EXPORTED_VOLUMES}"
        sudo umount -f ${EXPORTED_VOLUMES} 2>/dev/null
    fi

    # Disable the DM device so that the space for the loopback device is really freed
    # when we remove SERVICED_VOLUMES_PATH
    echo "Cleaning up serviced storage ..."
    sudo ${SERVICED_STORAGE} -v disable ${SERVICED_VOLUMES_PATH}

    echo "Removing up ${TEST_VAR_PATH} ..."
    sudo rm -rf ${TEST_VAR_PATH}
    echo "Finished test cleanup"
}
