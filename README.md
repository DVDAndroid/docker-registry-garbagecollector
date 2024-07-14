docker-registry-garbagecollector
===

This script cleanups the docker registry by removing tags that are not referenced by any image.

Meant to be used with [docker-registry-ui](https://github.com/Joxit/docker-registry-ui/).

# Usage

Refer to [sample.compose.yaml](sample.compose.yaml) for a sample docker-compose configuration.

Refer to [sample.config.yaml](sample.config.yaml) for a sample configuration file of docker-registry.

# Explanation

The script will:

1. catch all "delete" actions from docker-registry
2. runs `registry garbage-collect config.yml` into the docker-registry container
3. if there are 0 tags left, it will remove the repository from the filesystem
