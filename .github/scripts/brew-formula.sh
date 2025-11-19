#!/bin/bash

set -euo pipefail

tmpl=${1:?template parameter is required}
tag=${2:?tag parameter is required}
env_file=.env

export SELEBROW_VERSION=${tag#v}

env=$(gh release view "$tag" --json assets --jq '.assets.[] | "export \(.name | ascii_upcase | gsub("[-.]"; "_"))=\(.digest | split(":")[1])"')
eval "$env"
envsubst < "$tmpl"
