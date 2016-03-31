#######################################################
#
# Control Center Smoke Test
#
# Please add any tests you want executed at the bottom.
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
SERVICED_VARPATH=/tmp/serviced-smoke/var
IP=$(ip addr show docker0 | grep -w inet | awk {'print $2'} | cut -d/ -f1)
HOSTNAME=$(hostname)

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
            fail "ERROR: docker inspect "${wget_image}" is not available - wget tests will fail"
       fi
    fi
}

# Add the vhost to /etc/hosts so we can resolve it for the test
add_to_etc_hosts() {
    if [ -z "$(grep -e "^${IP} websvc.${HOSTNAME}" /etc/hosts)" ]; then
        sudo /bin/bash -c "echo ${IP} websvc.${HOSTNAME} >> /etc/hosts"
    fi
}

cleanup() {
    # remove the service to free up the disk space allocated in the devicemapper pool
    echo "Removing testsvc (if any) ..."
    sudo ${SERVICED} service remove testsvc

    echo "Stopping serviced ..."
    sudo pkill -9 serviced

    echo "Removing all docker containers ..."
    docker ps -qa | xargs --no-run-if-empty docker rm -fv

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


start_serviced() {
    # Note that we have to set SERVICED_MASTER instead of using the -master command line arg
    #   all of to force the proper subdirectories to be created under SERVICED_VARPATH
    echo "Starting serviced..."
    mkdir -p ${SERVICED_VARPATH}
    sudo GOPATH=${GOPATH} PATH=${PATH} SERVICED_VARPATH=${SERVICED_VARPATH} SERVICED_MASTER=1 ${SERVICED} --allow-loop-back=true --agent server &

    echo "Waiting 120 seconds for serviced to become the leader..."
    retry 180 wget --no-check-certificate http://${HOSTNAME}:443 -O- &>/dev/null
    return $?
}

# Add a host
add_host() {
    HOST_ID=$(${SERVICED} host add "${IP}:4979" default)
    sleep 1
    [ -z "$(${SERVICED} host list ${HOST_ID} 2>/dev/null)" ] && return 1
    return 0
}

add_template() {
    TEMPLATE_ID=$(${SERVICED} template compile ${DIR}/dao/testsvc | ${SERVICED} template add)
    sleep 1
    [ -z "$(${SERVICED} template list ${TEMPLATE_ID})" ] && return 1
    return 0
}

deploy_service() {
    echo "Deploying template id '${TEMPLATE_ID}'"
    echo ${SERVICED} template deploy ${TEMPLATE_ID} default testsvc
    SERVICE_ID=$(${SERVICED} template deploy ${TEMPLATE_ID} default testsvc)
    echo " deployed template id '${TEMPLATE_ID}' - SERVICE_ID='${SERVICE_ID}'"
    sleep 2
    [ -z "$(${SERVICED} service list ${SERVICE_ID})" ] && return 1
    return 0
}

start_service() {
    ${SERVICED} service start ${SERVICE_ID}
    sleep 5
    [[ "1" == $(serviced service list ${SERVICE_ID} | python -c "import json, sys; print json.load(sys.stdin)['DesiredState']") ]] || return 1
    return 0
}

stop_service() {
    ${SERVICED} service stop ${SERVICE_ID}
    sleep 10
    [[ "0" == $(serviced service list ${SERVICE_ID} | python -c "import json, sys; print json.load(sys.stdin)['DesiredState']") ]] || return 1
    return 0
}

test_started() {
    ./smoke.py started
    rc=$?
    return $rc
}

test_vhost() {
    wget --no-check-certificate -qO- https://websvc.${HOSTNAME} &>/dev/null || return 1
    return 0
}

test_service_port() {

    # make sure it is accessible
    wget -qO- http://${HOSTNAME}:1234 || return 1

    # make sure it is accessible via ipv6
    # wget --no-check-certificate -qO- http://[${IP6}]:1234 || return 1
}

test_assigned_ip() {
    echo "Testing assigned IP at ${IP}:1000"
    docker run zenoss/ubuntu:wget /bin/bash -c "wget ${IP}:1000 -qO- &>/dev/null" || return 1
    return 0
}

test_config() {
    echo "Testing configuration file at ${IP}:1000/etc/my.cnf"
    docker run zenoss/ubuntu:wget /bin/bash -c "wget --no-check-certificate -qO- ${IP}:1000/etc/my.cnf | grep 'innodb_buffer_pool_size'"  || return 1
    return 0
}

test_dir_config() {
    [ "$(wget --no-check-certificate -qO- https://websvc.${HOSTNAME}/etc/bar.txt)" == "baz" ] || return 1
    return 0
}

test_attached() {
    varx=$(${SERVICED} service attach s1 whoami | tr -d '\r')
    if [[ "$varx" == "root" ]]; then
        return 0
    fi
    return 1
}

test_port_mapped() {
    echo "${SERVICED} service attach s1 wget -qO- http://localhost:9090/etc/bar.txt"
    varx=`${SERVICED} service attach s1 wget -qO- http://localhost:9090/etc/bar.txt 2>/dev/null | tr -d '\r'| grep -v "^$" | tail -1`
    if [[ "$varx" == "baz" ]]; then
        return 0
    fi
    return 1
}

test_snapshot() {
    SNAPSHOT_ID=$(${SERVICED} service snapshot testsvc)
    ${SERVICED} snapshot list | grep -q ${SNAPSHOT_ID}
    return $?
}

test_snapshot_errs() {
    # make sure snapshot add returns non-zero code on error
    ${SERVICED} snapshot add invalid-id &>/dev/null
    if [[ "$?" == 0 ]]; then
        return 1
    fi

    # make sure service snapshot returns non-zero code on error
    ${SERVICED} service snapshot invalid-id &>/dev/null
    if [[ "$?" == 0 ]]; then
        return 1
    fi

    # make sure snapshot rollback returns non-zero code on error
    ${SERVICED} snapshot rollback invalid-id &>/dev/null
    if [[ "$?" == 0 ]]; then
        return 1
    fi

    return 0
}

test_service_shell() {
    sentinel=smoke_test_service_shell_sentinel_$$
    container=smoke_test_service_shell_$$
    ${SERVICED} service shell -s=$container s1 echo $sentinel
    docker logs $container | grep -q $sentinel
    return $?
}

test_service_run() {
    set -x

    # Make sure we start with no snapshots, othewise the checks below may pass for the wrong reason
    SNAPSHOT_COUNT=`${SERVICED} service list-snapshots s2 | wc -l`
    [ "${SNAPSHOT_COUNT}" == "0" ] || return 1

    local rc=""
    ${SERVICED} service run s2 exit0; rc="$?"
    [ "$rc" -eq 0 ] || return "$rc"

    # Verify that the commit moved the 'latest' tag to the same layer ID as the snapshot
    TENANT_ID=`${SERVICED} service status testsvc --show-fields ServiceID | grep -v ServiceID `
    LATEST_IMAGE_ID=`docker images | grep ${TENANT_ID} | grep latest | awk '{print $3}' `
    SNAPSHOT_LABEL=`${SERVICED} service list-snapshots s2 | cut -d_ -f2 `
    SNAPSHOT_IMAGE_ID=`docker images | grep ${SNAPSHOT_LABEL} | awk '{print $3}' `
    [ "${SNAPSHOT_IMAGE_ID}" == "${LATEST_IMAGE_ID}" ] || return 1

    ${SERVICED} service run s2 exit1; rc="$?"
    [ "$rc" -eq 42 ] || return "255"

    # Verify that the no additional snapshots were created
    SNAPSHOT_COUNT=`${SERVICED} service list-snapshots s2 | wc -l`
    [ "${SNAPSHOT_COUNT}" == "1" ] || return 1

    # make sure kills to 'runs' are working OK
    for signal in INT TERM; do
        local sleepyPid=""
        ${SERVICED} service run s2 sleepy60 &
        sleepyPid="$!"
        sleep 10
        kill -"$signal" "$sleepyPid"
        sleep 10
        kill -0 "$sleepyPid" &>/dev/null && return 1 # make sure job is gone
    done
    set +x
}

test_service_run_command() {
    set -x
    local rc=""
    ${SERVICED} service run s1 commands-exit0; rc="$?"
    [ "$rc" -eq 0 ] || return "$rc"
    ${SERVICED} service run s1 commands-exit1; rc="$?"
    [ "$rc" -eq 42 ] || return "255"
    ${SERVICED} service run s1 commands-exit0-nc; rc="$?"
    [ "$rc" -eq 0 ] || return "$rc"
    ${SERVICED} service run s1 commands-exit1-nc; rc="$?"
    [ "$rc" -eq 42 ] || return "255"
    # make sure kills to 'commands' are working OK
    for signal in INT TERM; do
        local sleepyPid=""
        ${SERVICED} service run s1 commands-sleepy60 &
        sleepyPid="$!"
        sleep 10
        kill -"$signal" "$sleepyPid"
        sleep 10
        kill -0 "$sleepyPid" &>/dev/null && return 1 # make sure job is gone
    done
    for signal in INT TERM; do
        local sleepyPid=""
        ${SERVICED} service run s1 commands-sleepy60-nc &
        sleepyPid="$!"
        sleep 10
        kill -"$signal" "$sleepyPid"
        sleep 10
        kill -0 "$sleepyPid" &>/dev/null && return 1 # make sure job is gone
    done
    set +x
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

# Force a clean environment
echo "Starting Pre-test cleanup ..."
cleanup --ignore-errors
echo "Pre-test cleanup complete"


# Setup
install_prereqs
add_to_etc_hosts

# Run all the tests
start_serviced             && succeed "Serviced became leader within timeout"    || fail "serviced failed to become the leader within 120 seconds."
retry 20 add_host          && succeed "Added host successfully"                  || fail "Unable to add host"
add_template               && succeed "Added template successfully"              || fail "Unable to add template"
deploy_service             && succeed "Deployed service successfully"            || fail "Unable to deploy service"
test_service_run           && succeed "Service run ran successfully"             || fail "Unable to run service run"
start_service              && succeed "Started service"                          || fail "Unable to start service"
retry 10 test_started      && succeed "Service containers started"               || fail "Unable to see service containers"

retry 10 test_vhost        && succeed "VHost is up and listening"                || fail "Unable to access service VHost"

retry 10 test_assigned_ip  && succeed "Assigned IP is listening"                 || fail "Unable to access service by assigned IP"
retry 10 test_config       && succeed "Config file was successfully injected"    || fail "Unable to access config file"

retry 10 test_dir_config   && succeed "-CONFIGS- file was successfully injected"   || fail "Unable to access -CONFIGS- file"

retry 10 test_attached     && succeed "Attached to container"                      || fail "Unable to attach to container"
retry 10 test_port_mapped  && succeed "Attached and hit imported port correctly"   || fail "Unable to connect to endpoint"
test_snapshot              && succeed "Created snapshot"                           || fail "Unable to create snapshot"
test_snapshot_errs         && succeed "Snapshot errs returned expected err code"   || fail "Snapshot errs did not return expected err code"
test_service_shell         && succeed "Service shell ran successfully"             || fail "Unable to run service shell"
test_service_port          && succeed "Accessing public endpoint via port success" || fail "Unable to access public endpoint via port"
stop_service               && succeed "Stopped service"                            || fail "Unable to stop service"
# "trap cleanup EXIT", above, will handle cleanup
