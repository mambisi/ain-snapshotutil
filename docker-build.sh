#!/bin/bash

for FILE in ./docker/*/Dockerfile
do
    tag=$(basename "$(dirname "$FILE")")
    dockerfile=$FILE
    echo "$tag"
    echo "$dockerfile"
    docker build -f "$dockerfile" -t $tag
    docker tag "$tag" "$HOST/$PROECTID/$tag"
done