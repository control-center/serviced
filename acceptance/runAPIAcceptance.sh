#!/bin/bash
#
# Runs the docker container in which cucumber executes.
# See usage statement below for more details
#
# NOTE: To pass options to cucumber, you must set the CUCUMBER_OPTS
#       environment variable. For example,
#       $ CUCUMBER_OPTS="--name MyFeature" ./dockerRun.sh
#

#
# Set defaults
#
debug=false
interactive=false
TAGS=()
DATASET=default
VERSION=$(cat dockerImage/VERSION)

set -e

while (( "$#" )); do
    if [ "$1" == "-u" ]; then
        APPLICATION_USERID="${2}"
        shift 2
    elif [ "$1" == "-p" ]; then
        APPLICATION_PASSWORD="${2}"
        shift 2
    elif [ "$1" == "-a" ]; then
        APPLICATION_URL="${2}"
        shift 2
    elif [ "$1" == "--tags" ]; then
        TAGS+=("${2}")
        shift 2
    elif [ "$1" == "--dataset" ]; then
        DATASET="${2}"
        shift 2
    elif [ "$1" == "--debug" ]; then
        debug=true
        shift 1
    elif [ "$1" == "-i" ]; then
        interactive=true
        shift 1
    else
        if [ "$1" != "-h" -a "$1" != "--help" ]; then
            echo "ERROR: invalid argument '$1'"
            echo "Try '$0 --help' for more information"
            exit 1
        fi
        echo "USAGE: runUIAcceptance.sh.sh [-a url] [-u userid] [-p password]"
        echo "       [--tags tagname [--tags tagname]]"
        echo "       [--dataset setName] [--debug] [-i] [-h, --help]"
        echo ""
        echo "where"
        echo "    -a url                the URL of the serviced application"
        echo "    -u userid             a valid seviced user id (required)"
        echo "    -p password           the password for userid (required)"
        echo "    --tags tagname        specifies a Cucumber tag"
        echo "    --dataset setName     identifies the dataset to use"
        echo "    --debug               opens the browser on the host's DISPLAY"
        echo "    -i                    interactive mode. Starts a bash shell with all of the same"
        echo "                          env vars but doesn't run anything"
        echo "    -h, --help             print this usage statement and exit"
        exit 1
    fi

done

if [ -z "${APPLICATION_URL-}" ]; then
    echo "ERROR: URL undefined. You must either set the environment variable"
    echo "       APPLICATION_URL, or specify it with the -a command line arg"
    exit 1
fi

if [ -z "${APPLICATION_USERID-}" ]; then
    echo "ERROR: userid undefined. You must either set the environment variable"
    echo "       APPLICATION_USERID, or specify it with the -u command line arg"
    exit 1
fi

if [ -z "${APPLICATION_PASSWORD-}" ]; then
    echo "ERROR: password undefined. You must either set the environment variable"
    echo "       APPLICATION_PASSWORD, or specify it with the -p command line arg"
    exit 1
fi

if [ -z "${SERVICED_ETC_PATH}" ]; then
    if [ -z "${SERVICED_HOME}" ]; then
        echo "ERROR: Both SERVICED_HOME and SERVICED_ETC_PATH are undefined."
        exit 1
    fi
    SERVICED_ETC_PATH=${SERVICED_HOME}/etc
fi

if [ -z "${SERVICED_ISVCS_PATH}" ]; then
    if [ -z "${SERVICED_HOME}" ]; then
        echo "ERROR: Both SERVICED_HOME and SERVICED_ISVCS_PATH are undefined."
        exit 1
    fi
    SERVICED_ISVCS_PATH=${SERVICED_HOME}/var/isvcs
fi

#
# Get the current UID and GID. These are passed into the container for use in
# creating a container-local user account so ownership of files created in the
# container will match the user in the host OS.
#
CALLER_UID=`id -u`
CALLER_GID=`id -g`

if [ -n "${TAGS}" ]; then
    for i in "${TAGS[@]}"
    do
        CUCUMBER_OPTS="${CUCUMBER_OPTS} --tags $i"
    done
fi

if [ -n "${CUCUMBER_OPTS}" ]; then
    echo "CUCUMBER_OPTS=${CUCUMBER_OPTS}"
fi

if [ "$interactive" == true ]; then
    INTERACTIVE_OPTION="-i"
    CMD="bash"
elif [ `uname -s` == "Darwin" ]; then
    # FIXME: This may work with a little testing and tweaking
    echo "ERROR: not supported on Mac OS X"
    exit 1
else
    CMD="runCucumber.sh ${CUCUMBER_OPTS}"
fi

parse_host() {
    # Get the URL by removing the protocol
    url="$(echo $1 | sed -e's,^\(.*://\)\(.*\),\2,g')"
    # Get the host (in a simple-minded way)
    host="$(echo $url | cut -d/ -f1)"
    echo "$host"
}

# Lookup host IP from name. Use ping because we can't assume that nslookup is installed.
lookup_host_ip() {
    PING_RESULT=`ping -c 1 $1 2>&1 | grep ^PING `
    if [ -z "${PING_RESULT}" ]; then
        echo "ERROR: can't find target host IP address: 'ping -c 1 $1' failed" 1>&2
        exit 1
    fi

    host_ip="$(echo ${PING_RESULT} | awk '{print $3}' | sed -r 's/\(//' | sed -r 's/\)//')"
    echo "$host_ip"
}

TARGET_HOST=`parse_host ${APPLICATION_URL}`
TARGET_HOST_IP=`lookup_host_ip ${TARGET_HOST}`

HOST_IP=`hostname -i`
if [[ $HOST_IP == 127* ]]; then
    # serviced won't allow us to add a loopback address, so use docker's as an alternative
    echo "Overriding default HOST_IP ($HOST_IP)"
    HOST_IP=$(ip addr show docker0 | grep -w inet | awk {'print $2'} | cut -d/ -f1)
fi
echo "Using HOST_IP=$HOST_IP"

#
# If we're not running on Ubuntu 14.04 (the same version as the zenoss/capybara image), then
# we need to mount the host's libdevmapper into the container since the container
# doesn't have a compatible library
#
MOUNT_DEVMAPPER=false
if [ -f /etc/redhat-release ]; then
   MOUNT_DEVMAPPER=true
else
   UBUNTU_RELEASE=`lsb_release -r | cut -f2`
   if [ "$UBUNTU_RELEASE" != "14.04" ]; then
      MOUNT_DEVMAPPER=true
   fi
fi

LIB_DEVMAPPER_MOUNT=""
if [ "$MOUNT_DEVMAPPER" == "true" ]; then
    LIBDEVMAPPER_SOURCE=`ldd ../serviced | grep libdevmapper | awk '{print $3}'`
    LIBDEVMAPPER_TARGET=`echo $LIBDEVMAPPER_SOURCE | rev | cut -d '/' -f1 | rev`
    LIB_DEVMAPPER_MOUNT="-v ${LIBDEVMAPPER_SOURCE}:/lib/x86_64-linux-gnu/${LIBDEVMAPPER_TARGET}"
fi

#
# If not running in a jenkins job, provide default values for build number and job name
#
if [ -z "$BUILD_NUMBER" ]; then
    BUILD_NUMBER="local-build"
fi
if [ -z "$JOB_NAME" ]; then
    JOB_NAME="serviced-api-acceptance"
fi

cp -u `pwd`/../serviced `pwd`/api
trap 'docker rm -f api_acceptance' INT

docker run --rm --name api_acceptance \
    --add-host=${TARGET_HOST}:${TARGET_HOST_IP} \
    -v /tmp/.X11-unix:/tmp/.X11-unix:ro \
    ${DEBUG_OPTION} \
    -v `pwd`/api:/capybara:rw \
    -v `pwd`/common:/common:rw \
    -v ${SERVICED_ETC_PATH}:/opt/serviced/etc \
    -v ${SERVICED_ISVCS_PATH}:/opt/serviced/var/isvcs \
    ${LIB_DEVMAPPER_MOUNT} \
    -e CALLER_UID=${CALLER_UID} \
    -e CALLER_GID=${CALLER_GID} \
    -e BUILD_NUMBER=${BUILD_NUMBER} \
    -e JOB_NAME=${JOB_NAME} \
    -e DATASET=${DATASET} \
    -e APPLICATION_URL=${APPLICATION_URL} \
    -e APPLICATION_USERID=${APPLICATION_USERID} \
    -e APPLICATION_PASSWORD=${APPLICATION_PASSWORD} \
    -e HOST_IP=${HOST_IP} \
    -e TARGET_HOST=${TARGET_HOST} \
    ${INTERACTIVE_OPTION} \
    -t zenoss/capybara:${VERSION}-xenial \
    ${CMD}
