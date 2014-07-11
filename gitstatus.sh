#!/usr/bin/env bash

set -e
GITCOMMIT=`git rev-parse --short HEAD`

#if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
if [ -n "$(git status --porcelain)" ]; then
        GITCOMMIT="$GITCOMMIT-dirty"
fi

echo $GITCOMMIT

