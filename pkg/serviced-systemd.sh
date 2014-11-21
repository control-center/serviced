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
	while pgrep -fla "bin/serviced$|bin/serviced "; do
	    sleep 5
	done

	# wait for serviced to stop listening
	echo "$(date): waiting for serviced to stop listening"
	while netstat -plant | grep 'LISTEN.*/serviced$'; do
		sleep 2
	done

	echo "$(date): serviced is now stopped - done with post-stop"
	rc=0
	;;
  *)
        echo $"Usage: $0 {pre-start|post-stop}"
        exit 2
esac

exit $rc
