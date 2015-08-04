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
runAsRoot=false
DRIVER_NAME=selenium
TIMEOUT=10
DATASET=default

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
    elif [ "$1" == "-d" ]; then
        DRIVER_NAME="${2}"
        shift 2
    elif [ "$1" == "-t" ]; then
        TIMEOUT="${2}"
        shift 2
    elif [ "$1" == "--dataset" ]; then
        DATASET="${2}"
        shift 2
    elif [ "$1" == "--debug" ]; then
        debug=true
        shift 1
    elif [ "$1" == "--root" ]; then
        runAsRoot=true
        shift 1
    elif [ "$1" == "-i" ]; then
        interactive=true
        shift 1
    else
        if [ "$1" != "-h" ]; then
            echo "ERROR: invalid argument '$1'"
        fi
        echo "USAGE: runUIAcceptance.sh.sh [-a url] [-u userid] [-p password]"
        echo "       [-d driverName] [-t timeout] [--dataset setName]"
        echo "       [--debug] [--root] [-i] [-h]"
        echo ""
        echo "where"
        echo "    -a url                the URL of the serviced application"
        echo "    -u userid             a valid seviced user id (required)"
        echo "    -p password           the password for userid (required)"
        echo "    -d driverName         identifies the Capybara driver to use"
        echo "                          (e.g. selenium, selenium_chrome or poltergeist)"
        echo "    -t timeout            identifies the Capybara timeout to use (in seconds)"
        echo "    --dataset setName     identifies the dataset to use"
        echo "    --debug               opens the browser on the host's DISPLAY"
        echo "    --root                run the tests as root in the docker container"
        echo "    -i                    interactive mode. Starts a bash shell with all of the same"
        echo "                          env vars but doesn't run anything"
        echo "    -h                    print this usage statement and exit"
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

#
# Get the current UID and GID. These are passed into the container for use in
# creating a container-local user account so ownership of files created in the
# container will match the user in the host OS.
#
CALLER_UID=`id -u`
CALLER_GID=`id -g`

if [ "$debug" == true ]; then
    DEBUG_OPTION="-e DISPLAY=unix$DISPLAY"
fi

if [ "$interactive" == true ]; then
    INTERACTIVE_OPTION="-i"
    CMD="bash"
elif [ "$runAsRoot" == true ]; then
    CMD="runCucumber.sh --root ${CUCUMBER_OPTS}"
elif [ `uname -s` == "Darwin" ]; then
    echo "ERROR: missing required argument '--root' for Mac OS X"
    exit 1
else
    CMD="runCucumber.sh ${CUCUMBER_OPTS}"
fi

HOSTNAME=`hostname -s`
HOST_IP=`hostname -i`

trap 'docker rm -f ui_acceptance' INT

docker run --rm --name ui_acceptance \
    --add-host=${HOSTNAME}:${HOST_IP} \
    -v /tmp/.X11-unix:/tmp/.X11-unix:ro \
    ${DEBUG_OPTION} \
    -v /etc/timezone:/etc/timezone:ro \
    -v /etc/localtime:/etc/localtime:ro \
    -v `pwd`/ui:/capybara:rw \
    -e CALLER_UID=${CALLER_UID} \
    -e CALLER_GID=${CALLER_GID} \
    -e CAPYBARA_DRIVER=${DRIVER_NAME} \
    -e CAPYBARA_TIMEOUT=${TIMEOUT} \
    -e DATASET=${DATASET} \
    -e APPLICATION_URL=${APPLICATION_URL} \
    -e APPLICATION_USERID=${APPLICATION_USERID} \
    -e APPLICATION_PASSWORD=${APPLICATION_PASSWORD} \
    -e HOST_IP=${HOST_IP} \
    ${INTERACTIVE_OPTION} \
    -t zenoss/capybara:1.0.0 \
    ${CMD}
