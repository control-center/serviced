#!/usr/bin/env bash

docker ps | tail -n +2 | awk '{ print $1 }' | xargs docker kill 
docker ps -a | tail -n +2 | awk '{ print $1 }' | xargs docker rm
docker images | tail -n +2 | awk '{ print $1 }' | xargs docker rmi

