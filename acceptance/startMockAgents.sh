#!/bin/bash

set -e

cd ..
make -f makefile.mockAgent mockAgent
cd acceptance

echo `mockAgent/mockAgent --config-file ui/features/data/default/hosts.json --host defaultHost` &
echo `mockAgent/mockAgent --config-file ui/features/data/default/hosts.json --host host2` &
echo `mockAgent/mockAgent --config-file ui/features/data/default/hosts.json --host host3` &
echo `mockAgent/mockAgent --config-file ui/features/data/default/hosts.json --host host4` &
echo `mockAgent/mockAgent --config-file ui/features/data/default/hosts.json --host host5` &

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

while true; 
  do sleep 5m;
done
