#!/usr/bin/env bash

docker container prune
docker rm $(docker container ls -f name="redis" -q) --force
docker rm $(docker container ls -f name="remiro" -q) --force
docker network rm $(docker network ls -f name="e2e" -q)
docker volume rm $(docker volume ls -f name="e2e" -q)
docker image prune
# docker rmi $(docker images "remiro*" -q) --force
# docker rmi $(docker images "redis-rdb-tools*" -q) --force             
docker rmi hello-world

