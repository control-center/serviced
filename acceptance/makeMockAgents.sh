#!/bin/bash

set -e
trap 'kill $(jobs -p)' EXIT

cd ..
make mockAgent
cd acceptance
echo `mockAgent/mockAgent --config-file ui/features/data/default/hosts.json --host defaultHost` &
echo `mockAgent/mockAgent --config-file ui/features/data/default/hosts.json --host host2` &
echo `mockAgent/mockAgent --config-file ui/features/data/default/hosts.json --host host3` &

while true; do echo "CTRL C to stop"; sleep 5m; done
