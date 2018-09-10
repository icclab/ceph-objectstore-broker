#!/bin/bash
cf push $1 -f "manifest.yml" --vars-file="vars-file.yml"