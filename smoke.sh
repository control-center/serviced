#!/bin/bash
#######################################################
#
# Control Center Smoke Test
#
# Please add any tests you want executed at the bottom.
#
#######################################################

# Use a directory unique to this test to avoid collisions with other kinds of tests
TEST_VAR_PATH=/tmp/serviced-smoke/var
. test_lib.sh

# Add a host
add_host() {
    HOST_ID=$(sudo SERVICED_ISVCS_PATH=${SERVICED_ISVCS_PATH} SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} host add "${IP}:4979" default --register | tail -n 1)
    sleep 1
    [ -z "$(SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} host list ${HOST_ID} 2>/dev/null)" ] && return 1
    return 0
}

add_template() {
    TEMPLATE_ID=$(SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} template compile ${DIR}/dao/testsvc | SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} template add)
    sleep 1
    [ -z "$(SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} template list ${TEMPLATE_ID})" ] && return 1
    return 0
}

deploy_service() {
    echo "Deploying template id '${TEMPLATE_ID}'"
    echo SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} template deploy ${TEMPLATE_ID} default testsvc
    SERVICE_ID=$(SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} template deploy ${TEMPLATE_ID} default testsvc)
    echo " deployed template id '${TEMPLATE_ID}' - SERVICE_ID='${SERVICE_ID}'"
    sleep 2
    [ -z "$(SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service list ${SERVICE_ID})" ] && return 1
    return 0
}

start_service() {
    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service start ${SERVICE_ID}
    sleep 5
    [[ "1" == $(serviced service list ${SERVICE_ID} | python -c "import json, sys; print json.load(sys.stdin)['DesiredState']") ]] || return 1
    return 0
}

stop_service() {
    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service stop ${SERVICE_ID}
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
    echo "Testing vhost"
    wget --no-check-certificate --content-on-error -O- https://websvc.${HOSTNAME} || return 1
    return 0
}

test_service_port_http() {
    wget --no-check-certificate --content-on-error -O- http://${HOSTNAME}:1235 || return 1
}

test_service_port_tcp() {
    wget --no-check-certificate --content-on-error -O- http://${HOSTNAME}:1237 || return 1
}

test_service_port_https() {
    wget --no-check-certificate --content-on-error -O- https://${HOSTNAME}:1234 || return 1
}

test_service_port_tcp_tls() {
    wget --no-check-certificate --content-on-error -O- https://${HOSTNAME}:1236 || return 1
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
    varx=$(SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service attach s1 whoami | tr -d '\r')
    if [[ "$varx" == "root" ]]; then
        return 0
    fi
    return 1
}

test_port_mapped() {
    echo "SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service attach s1 wget -qO- http://localhost:9090/etc/bar.txt"
    varx=`SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service attach s1 wget -qO- http://localhost:9090/etc/bar.txt 2>/dev/null | tr -d '\r'| grep -v "^$" | tail -1`
    if [[ "$varx" == "baz" ]]; then
        return 0
    fi
    return 1
}

test_snapshot() {
    SNAPSHOT_ID=$(SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service snapshot testsvc)
    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} snapshot list | grep -q ${SNAPSHOT_ID}
    return $?
}

test_snapshot_errs() {
    # make sure snapshot add returns non-zero code on error
    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} snapshot add invalid-id &>/dev/null
    if [[ "$?" == 0 ]]; then
        return 1
    fi

    # make sure service snapshot returns non-zero code on error
    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service snapshot invalid-id &>/dev/null
    if [[ "$?" == 0 ]]; then
        return 1
    fi

    # make sure snapshot rollback returns non-zero code on error
    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} snapshot rollback invalid-id &>/dev/null
    if [[ "$?" == 0 ]]; then
        return 1
    fi

    return 0
}

test_service_shell() {
    sentinel=smoke_test_service_shell_sentinel_$$
    container=smoke_test_service_shell_$$
    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service shell -s=$container s1 echo $sentinel
    docker logs $container | grep -q $sentinel
    return $?
}

test_service_run() {
    set -x

    # Make sure we start with no snapshots, othewise the checks below may pass for the wrong reason
    SNAPSHOT_COUNT=`SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service list-snapshots s2 | wc -l`
    [ "${SNAPSHOT_COUNT}" == "0" ] || return 1

    local rc=""
    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service run s2 exit0; rc="$?"
    [ "$rc" -eq 0 ] || return "$rc"

    # Verify that the commit moved the 'latest' tag to the same layer ID as the snapshot
    TENANT_ID=`SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service status testsvc --show-fields ServiceID | grep -v ServiceID `
    LATEST_IMAGE_ID=`docker images | grep ${TENANT_ID} | grep latest | awk '{print $3}' `
    SNAPSHOT_LABEL=`SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service list-snapshots s2 | cut -d_ -f2 `
    SNAPSHOT_IMAGE_ID=`docker images | grep ${SNAPSHOT_LABEL} | awk '{print $3}' `
    [ "${SNAPSHOT_IMAGE_ID}" == "${LATEST_IMAGE_ID}" ] || return 1

    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service run s2 exit1; rc="$?"
    [ "$rc" -eq 42 ] || return "255"

    # Verify that the no additional snapshots were created
    SNAPSHOT_COUNT=`SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service list-snapshots s2 | wc -l`
    [ "${SNAPSHOT_COUNT}" == "1" ] || return 1

    # make sure kills to 'runs' are working OK
    for signal in INT TERM; do
        local sleepyPid=""
        SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service run s2 sleepy60 &
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
    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service run s1 commands-exit0; rc="$?"
    [ "$rc" -eq 0 ] || return "$rc"
    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service run s1 commands-exit1; rc="$?"
    [ "$rc" -eq 42 ] || return "255"
    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service run s1 commands-exit0-nc; rc="$?"
    [ "$rc" -eq 0 ] || return "$rc"
    SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service run s1 commands-exit1-nc; rc="$?"
    [ "$rc" -eq 42 ] || return "255"
    # make sure kills to 'commands' are working OK
    for signal in INT TERM; do
        local sleepyPid=""
        SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service run s1 commands-sleepy60 &
        sleepyPid="$!"
        sleep 10
        kill -"$signal" "$sleepyPid"
        sleep 10
        kill -0 "$sleepyPid" &>/dev/null && return 1 # make sure job is gone
    done
    for signal in INT TERM; do
        local sleepyPid=""
        SERVICED_ETC_PATH=${SERVICED_ETC_PATH} ${SERVICED} service run s1 commands-sleepy60-nc &
        sleepyPid="$!"
        sleep 10
        kill -"$signal" "$sleepyPid"
        sleep 10
        kill -0 "$sleepyPid" &>/dev/null && return 1 # make sure job is gone
    done
    set +x
}

###############################################################################
###############################################################################
#
# Test execution starts here
#
trap cleanup EXIT
print_env_info

# Force a clean environment
echo "Starting Pre-test cleanup ..."
cleanup --ignore-errors
echo "Pre-test cleanup complete"


# Setup
install_prereqs
add_to_etc_hosts

# Run all the tests
start_serviced             && succeed "Serviced has started within timeout"      || fail "serviced failed to start within $START_TIMEOUT seconds."
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

test_service_port_http     && succeed "Accessing public endpoint via HTTP port success"    || fail "Unable to access public endpoint via HTTP port"
test_service_port_https    && succeed "Accessing public endpoint via HTTPS port success"   || fail "Unable to access public endpoint via HTTPS port"
test_service_port_tcp      && succeed "Accessing public endpoint via TCP port success"     || fail "Unable to access public endpoint via TCP port"
test_service_port_tcp_tls  && succeed "Accessing public endpoint via TCP/TLS port success" || fail "Unable to access public endpoint via TCP/TLS port"

stop_service               && succeed "Stopped service"                            || fail "Unable to stop service"
# "trap cleanup EXIT", above, will handle cleanup
