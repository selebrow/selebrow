#!/bin/bash

if [ -z "$1" ]; then
    exit 1
fi

text=$(echo "$1" | tr '[:upper:]' '[:lower:]' | sed 's/[_ ]/-/g' | sed 's/[^a-z0-9\.-]//g')

echo "${text:0:64}"
