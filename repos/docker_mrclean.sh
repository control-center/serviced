#!/usr/bin/env bash


containers=`docker ps -q`
if [[ -n "$containers" ]]; then
	docker kill $containers
fi
containers=`docker ps -a -q`
if [[ -n "$containers" ]]; then
	docker rm `docker ps -a -q`
fi
containers=`docker images -q`
if [[ -n "$containers" ]]; then
	docker rmi `docker images -q`
fi

