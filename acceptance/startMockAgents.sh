#!/bin/bash
DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

wait_forever=true

while (( "$#" )); do
    if [ "$1" == "--no-wait" ]; then
         wait_forever=false
        shift 1
    else
        if [ "$1" != "-h" ]; then
            echo "ERROR: invalid argument '$1'"
        fi
        echo "USAGE: startMockAgents.sh [--no-wait]"
        echo ""
        echo "where"
        echo "    --no-wait    do not wait"
        exit 1
    fi

done

HOSTNAME=`hostname -s`

set -e

set -x
cd ${DIR}
${DIR}/mockAgent/mockAgent --config-file ${DIR}/ui/features/data/default/hosts.json --host defaultHost --address ${HOSTNAME} &
${DIR}/mockAgent/mockAgent --config-file ${DIR}/ui/features/data/default/hosts.json --host host2 --address ${HOSTNAME} &
${DIR}/mockAgent/mockAgent --config-file ${DIR}/ui/features/data/default/hosts.json --host host3 --address ${HOSTNAME} &
${DIR}/mockAgent/mockAgent --config-file ${DIR}/ui/features/data/default/hosts.json --host host4 --address ${HOSTNAME} &
${DIR}/mockAgent/mockAgent --config-file ${DIR}/ui/features/data/default/hosts.json --host host5 --address ${HOSTNAME} &
set +x

WAIT=0
while [ $WAIT -lt 5 ]; do
  echo "."
  sleep 1s
  let WAIT=WAIT+1
done

if jobs -l | grep -q "Done";
  then
    jobs 1>&2
    echo "Mock agent(s) failed, closing" 1>&2
    pkill mockAgent
    exit 1;
fi

echo "Mock agents up, test suite may be run"

while $wait_forever ;
  do sleep 5m;
done
