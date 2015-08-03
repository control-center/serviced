#!/bin/bash

interval=1


if [ ! -x /usr/bin/curl ]; then
	if ! apt-get install -y curl ;then
		echo "Could not install curl"
		exit 1
	fi

fi
echo "posting loadavg at $interval second(s) interval"
while :
	do
	now=`date +%s`
	value=`cat /proc/loadavg | cut -d ' ' -f 1`
	data="{\"control\":{\"type\":null,\"value\":null},\"metrics\":[{\"metric\":\"loadavg\",\"timestamp\":$now,\"value\":$value,\"tags\":{\"name\":\"value\"}}]}"

	output=`curl -s -XPOST -H "Content-Type: application/json" -d "$data" "$CONTROLPLANE_CONSUMER_URL"`

	if ! [[ "$output" == *OK* ]]
	then
		echo "failure";
	fi

	sleep $interval
done
