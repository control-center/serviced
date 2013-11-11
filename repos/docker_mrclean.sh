#!/usr/bin/env bash


containers=`docker ps -q`
if [[ -n "$containers" ]]; then
	docker kill $containers
fi
containers=`docker ps -a -q`
if [[ -n "$containers" ]]; then
	docker rm $containers 
fi
containers=`docker images -q`
if [[ -n "$containers" ]]; then
	docker rmi $containers
fi

