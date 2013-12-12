#!/usr/bin/env bash

echo "Stopping containers..."
containers=`docker ps -q`
if [[ -n "$containers" ]]; then
	docker kill $containers
fi
echo "Deleting containers..."
containers=`docker ps -a -q`
if [[ -n "$containers" ]]; then
	docker rm $containers 
fi
echo "Deleting images..."
containers=`docker images -q`
if [[ -n "$containers" ]]; then
	docker rmi $containers
fi

