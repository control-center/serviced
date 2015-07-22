#!/bin/bash

# allows arbitrary tasks to be performed as a specific
# user. Useful for performing work on a bind-mounted
# filesystem from inside a docker container

USERGROUP="guysgroup"
USERNAME="guy"
USERHOME=/home/"$USERNAME"

# ensure $UID_X and $GID_X are set
if [[ -z $UID_X ]]; then
    echo "Please specify environment variable UID_X (eg: -e UID_X=\$(id -u))"
    exit 1
elif [[ -z $GID_X ]]; then
    echo "Please specify environment variable GID_X (eg: -e GID_X=\$(id -g))"
    exit 1
fi

if [[ ${UID_X} -ne 0 ]]; then
    # create user and group
    groupadd -g "$GID_X" -r "$USERGROUP"
    mkdir -p "$USERHOME"
    useradd -u "$UID_X" -r -g "$USERGROUP" -d "$USERHOME" -s /bin/bash "$USERNAME"
    chown "$USERNAME":"$USERGROUP" "$USERHOME"

    # kick off task as user
    su - "$USERNAME" -c "$*" 
else
    $*
fi
