#!/bin/bash
for FILE in ./docker/*/Dockerfile
do
    tag=$(basename "$(dirname "$FILE")")
    dockerfile=$FILE
    echo "$tag"
    echo "$dockerfile"
    docker build -t $tag "$(dirname "$FILE")"
    docker tag "$tag" "$HOST/$PROJECTID/$tag"
done