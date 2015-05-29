#!/bin/bash
#
# Make the user and group to run cucumber inside the docker container such that
# the uid/gid match the caller outside of the container.
#
# The caller should set the env vars CALLER_UID and CALLER_GID to their uid/gid values
# before calling this script.
#
# Based on work from https://github.com/control-center/serviced/blob/develop/build/userdo.sh
#
USERGROUP="cuke"
USERNAME="cuke"
USERHOME=/home/"$USERNAME"

# ensure $CALLER_UID and $CALLER_GID are set
if [[ -z $CALLER_UID ]]; then
    echo "Please specify environment variable UID (eg: -e UID=\$(id -u))"
    exit 1
elif [[ -z $CALLER_GID ]]; then
    echo "Please specify environment variable GID (eg: -e GID=\$(id -g))"
    exit 1
fi

#
# create a user and group inside the container which have the same uid/gid as
# the user outside of the container
groupadd -o -g "$CALLER_GID" -r "$USERGROUP"
mkdir -p "$USERHOME"
useradd -o -u "$CALLER_UID" -r -g "$USERGROUP" -d "$USERHOME" -s /bin/bash "$USERNAME"
chown "$USERNAME":"$USERGROUP" "$USERHOME"
