#!/bin/bash
DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

wait_forever=true
DATASET=default

while (( "$#" )); do
    if [ "$1" == "--no-wait" ]; then
         wait_forever=false
        shift 1
    elif [ "$1" == "--dataset" ]; then
        DATASET="${2}"
        shift 2
    else
        if [ "$1" != "-h" ]; then
            echo "ERROR: invalid argument '$1'"
        fi
        echo "USAGE: startMockAgents.sh [--no-wait] [--dataset setName]"
        echo ""
        echo "where"
        echo "    --no-wait             do not wait"
        echo "    --dataset setName     identifies the dataset to use"
        exit 1
    fi

done

if [ ! -d ${DIR}/ui/features/data/${DATASET} ]; then
    echo "ERROR: directory '${DIR}/ui/features/data/${DATASET}' does not exist"
    exit 1
fi

HOST_IP=`hostname -i`
if [[ $HOST_IP == 127* ]]; then
    echo "Overriding default HOST_IP ($HOST_IP)"
    HOST_IP=$(ip addr show docker0 | grep -w inet | awk {'print $2'} | cut -d/ -f1)
fi
echo "Using HOST_IP=$HOST_IP"

set -e

set -x
cd ${DIR}
${DIR}/mockAgent/mockAgent --config-file ${DIR}/ui/features/data/${DATASET}/hosts.json --host defaultHost --address ${HOST_IP} &
${DIR}/mockAgent/mockAgent --config-file ${DIR}/ui/features/data/${DATASET}/hosts.json --host host2 --address ${HOST_IP} &
${DIR}/mockAgent/mockAgent --config-file ${DIR}/ui/features/data/${DATASET}/hosts.json --host host3 --address ${HOST_IP} &
${DIR}/mockAgent/mockAgent --config-file ${DIR}/ui/features/data/${DATASET}/hosts.json --host host4 --address ${HOST_IP} &
${DIR}/mockAgent/mockAgent --config-file ${DIR}/ui/features/data/${DATASET}/hosts.json --host host5 --address ${HOST_IP} &
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
