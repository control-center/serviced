#!/bin/bash

# Helper script used by systemd to perform some lifecycle actions

rc=0

# See how we were called.
case "$1" in
  pre-start)
	echo "$(date): starting serviced-isvcs daemon for $(hostid) - waiting for docker"
	while ! /usr/bin/docker ps; do date ; sleep 1 ; done
	echo "$(date): docker is now ready - done with pre-start"
	sleep 1s
	/sbin/ifconfig
	rc=0
	;;
  post-stop)
	echo "$(date): post-stopping serviced-isvcs daemon - waiting for serviced-isvcs to stop"

	# wait for serviced-isvcs daemon to stop
	echo "$(date): waiting for serviced-isvcs daemon to stop"
	while pgrep -fla "bin/serviced-isvcs"; do
	    sleep 5
	done

	# wait for serviced to stop listening
	echo "$(date): waiting for serviced-isvcs to stop listening"
	while netstat -plant | grep 'LISTEN.*/serviced-isvcs$'; do
		sleep 2
	done

	# stop potentially running isvcs even though serviced stopped (column 2 is IMAGEID)
	echo "$(date): waiting for serviced isvcs to stop"
	docker ps --no-trunc | awk '$2 ~ /^zenoss\/serviced-isvcs/{print $1}' | while read c; do docker stop $c; done

	echo "$(date): serviced-isvcs is now stopped - done with post-stop"
	rc=0
	;;
  start)
	echo "$(date): starting serviced-isvcs daemon for $(hostid)"
	export SERVICED_MASTER=0
	export SERVICED_AGENT=0
	exec $(dirname -- $0)/$(basename -- $0 .sh)
	;;
  *)
        echo $"Usage: $0 {pre-start|post-stop}"
        exit 2
esac

exit $rc
