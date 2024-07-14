!#/bin/bash

docker build -t docker-registry-garbagecollector:latest .
docker tag docker-registry-garbagecollector:latest registry.docker.lan/docker-registry-garbagecollector:latest
docker tag docker-registry-garbagecollector:latest registry.docker.lan/docker-registry-garbagecollector:1
docker tag docker-registry-garbagecollector:latest registry.docker.lan/docker-registry-garbagecollector:1.0
docker tag docker-registry-garbagecollector:latest registry.docker.lan/docker-registry-garbagecollector:1.0.0
docker push registry.docker.lan/docker-registry-garbagecollector:latest
docker push registry.docker.lan/docker-registry-garbagecollector:1
docker push registry.docker.lan/docker-registry-garbagecollector:1.0
docker push registry.docker.lan/docker-registry-garbagecollector:1.0.0
