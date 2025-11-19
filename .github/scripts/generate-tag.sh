#!/bin/bash

VERSION_FILE=VERSION.txt
RELEASE_TAG_PREFIX=v

VERSION_SOURCE=$(cat $VERSION_FILE)
MAJOR_VERSION=$(echo $VERSION_SOURCE | cut -d'.' -f1)
MINOR_VERSION=$(echo $VERSION_SOURCE | cut -d'.' -f2)

git fetch -q origin --prune --prune-tags
TAGS_COUNT=$(git tag | wc -l)      
if [ "$TAGS_COUNT" != "0" ] ; then
    LATEST_TAG=$(git tag | sort -rV | grep -E "^${RELEASE_TAG_PREFIX}${MAJOR_VERSION}.${MINOR_VERSION}.[0-9]{1,}" | head -1 || echo "0")        
    if [ "$LATEST_TAG" == "0" ] ; then
        PATCH_VERSION="0"
    else
        CUR_PATCH_VERSION=$(echo $LATEST_TAG | cut -d'.' -f3)
        PATCH_VERSION=`echo $((CUR_PATCH_VERSION + 1))`
    fi
else
    PATCH_VERSION="0"
fi

RELEASE_VERSION="${RELEASE_TAG_PREFIX}${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_VERSION}"
echo $RELEASE_VERSION
