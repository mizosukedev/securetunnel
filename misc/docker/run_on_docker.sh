#!/bin/sh

docker run                         \
  --rm                             \
  -v "$(pwd)":"/go/src/${PWD##*/}" \
  -w "/go/src/${PWD##*/}"          \
  -u "$(id -u):$(id -g)"           \
  -e HOME=/tmp                     \
  golang:1.18                      \
  "$@"
