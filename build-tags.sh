#!/usr/bin/env bash

# Determine the build tags necessary to build serviced, based on the builder's environment

# We always need daemon, so serviced will build btrfs and devicemapper stuff
BUILD_TAGS="daemon"

# Use the root tag if we're root, so we can run rooty tests
if [ $UID -eq 0 ]; then
    BUILD_TAGS+=" root"
fi

# test whether "libdevmapper.h" is new enough to support deferred remove functionality.
if 
    command -v gcc &> /dev/null \
    && ! ( echo -e  '#include <libdevmapper.h>\nint main() { dm_task_deferred_remove(NULL); }'| gcc -ldevmapper -xc - &> /dev/null ) \
; then
       BUILD_TAGS+=' libdm_no_deferred_remove'
fi

echo ${BUILD_TAGS}
