#!/bin/bash

# Helper script used by systemd to perform some lifecycle actions

rc=0

# See how we were called.
case "$1" in
    pre-start)
        echo "$(date): starting serviced daemon for $(hostid) - waiting for docker"
        while ! /usr/bin/docker ps; do date ; sleep 1 ; done
        echo "$(date): docker is now ready - done with pre-start"
        sleep 1s
        /sbin/ifconfig
        rc=0
        ;;
    post-stop)
        echo "$(date): post-stopping serviced daemon - waiting for serviced to stop"
        # wait for serviced daemon to stop
        echo "$(date): waiting for serviced daemon to stop"
        while pgrep -fla "bin/serviced\s+"; do
            sleep 5
        done
        
        # wait for serviced to stop listening
        echo "$(date): waiting for serviced to stop listening"
        while netstat -plant | grep 'LISTEN.*/serviced$'; do
            sleep 2
        done
        
        # stop and remove potentially running isvcs even though serviced stopped (column 2 is IMAGEID)
        echo "$(date): waiting for serviced isvcs to stop"
        for i in $(serviced version | grep IsvcsImages | awk -F'[][]' '{print $2}'); do
            docker ps -a | awk '$2 == "'"$i"'" {print $1}' | xargs docker rm -f 2> /dev/null
        done
        echo "$(date): serviced is now stopped - done with post-stop"
        rc=0
        ;;
    *)
        echo $"Usage: $0 {pre-start|post-stop}"
        exit 2
esac
exit $rc
