#!/bin/bash

if [ $# = 0 ]; then
    echo "Please provide name of output executable"
    exit
fi

CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $1 .