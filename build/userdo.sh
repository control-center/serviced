#!/bin/bash

# allows arbitrary tasks to be performed as a specific
# user. Useful for performing work on a bind-mounted
# filesystem from inside a docker container

USERGROUP="guysgroup"
USERNAME="guy"
USERHOME=/home/"$USERNAME"

# ensure $UID_X and $GID_X are set
if [[ -z $UID_X ]]; then
    echo "Please specify environment variable UID (eg: -e UID=\$(id -u))"
    exit 1
elif [[ -z $GID_X ]]; then
    echo "Please specify environment variable GID (eg: -e GID=\$(id -g))"
    exit 1
fi

# create user and group
groupadd -g "$GID_X" -r "$USERGROUP"
mkdir -p "$USERHOME"
useradd -u "$UID_X" -r -g "$USERGROUP" -d "$USERHOME" -s /bin/bash "$USERNAME"
chown "$USERNAME":"$USERGROUP" "$USERHOME"

# kick off task as user
su - "$USERNAME" -c "$*"
