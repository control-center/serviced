#!/usr/bin/env bash

set -e

usage() {
    >&2 echo "Usage: create-device-thinpool.sh (docker|serviced) /dev/XXX [/dev/YYY...]"
    exit 1
}

if [ $# -lt 2 ]; then
    usage
fi

GROUP="$1"
shift
DEVICES="$@"
POOL=${GROUP}-pool
META=${POOL}meta

if [ "${GROUP}" != "docker" -a "${GROUP}" != "serviced" ]; then
    usage
fi

# First create the physical device if necessary
for DEVICE in ${DEVICES}; do
    pvs ${DEVICE} || pvcreate ${DEVICE}
done

# Create the volume group
vgcreate ${GROUP} ${DEVICES}

# Create the metadata lv, 0.1% of the total space
SIZE=$(vgs --noheadings --nosuffix --units s -o vg_size ${GROUP})
META_SIZE=$((${SIZE} / 1000 + 1))
lvcreate -y -L ${META_SIZE}s -n ${META} ${GROUP}

# Create the data lv, 90% of the remaining free space, rounded to nearest 512
FREE_SPACE=$(vgs --noheadings --nosuffix --units b -o vg_free ${GROUP})
DATA_SIZE=$((${FREE_SPACE} * 90 / 100 / 512 * 512))
lvcreate -y -L ${DATA_SIZE}b -n ${POOL} ${GROUP}

# Create the thin pool
lvconvert -y --zero n --thinpool ${GROUP}/${POOL} --poolmetadata ${GROUP}/${META}

# docker expects device mapper device and not lvm device. Do the conversion.
lvs --nameprefixes --noheadings -o lv_name,kernel_major,kernel_minor ${GROUP} | while read line; do
    eval $line
    if [ "$LVM2_LV_NAME" = "${POOL}" ]; then
        TPOOL_NAME="/dev/mapper/$(cat /sys/dev/block/${LVM2_LV_KERNEL_MAJOR}:${LVM2_LV_KERNEL_MINOR}/dm/name)-tpool"
        case $GROUP in 
            docker)
                echo "Please add the following to /etc/default/docker, stop Docker, delete /var/lib/docker, and start it again:"
                echo
                echo DOCKER_OPTS=\"\${DOCKER_OPTS} --storage-opt dm.thinpooldev=${TPOOL_NAME}\"
                echo 
                ;;
            serviced)
                echo "Please add the following to /etc/default/serviced:"
                echo
                echo SERVICED_DM_THINPOOLDEV=\"${TPOOL_NAME}\"
                echo 
                ;;
        esac
    fi
done
