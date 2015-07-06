#!/usr/bin/env bash

# Converts a serviced-used Docker install with a btrfs storage driver to use devicemapper instead.

fail () {
    echo "!!! Unable to convert: $@" >&2
    exit 1
}

log () {
    echo " *  $@" >&2
}

confirm () {
    read -r -p "${1:-Continue? [y/N]} " response
    case $response in
        [yY][eE][sS]|[yY]) 
            true
            ;;
        *)
            false
            ;;
    esac
}

# Verify we have root permission.
if [[ $EUID -ne 0 ]]; then
   fail This script must be run as root
fi

# Figure out init system for start/stop Docker
if [ -d /usr/lib/systemd ]; then
    CTL_CMD="systemctl"
elif [ -d /usr/share/upstart ]; then
    CTL_CMD=""
else
    fail Unable to determine init system
fi

# Verify /var/lib/docker is on its own partition with a btrfs filesystem
MOUNT_LINE="$(mount -t btrfs | grep /var/lib/docker)"
if [ -z "${MOUNT_LINE}" ]; then
    fail /var/lib/docker is not a separate Btrfs partition.
fi

echo IMPORTANT: This script will destroy your current Docker storage directory.
echo This will cause all Docker images currently on your system to be lost.  
echo 
echo By default, this script will save the internal services image required by
echo Control Center, requiring approximately 600MB in /tmp.  If you have
echo a connection to the Internet, you are not required to save this image. Control
echo Center will automatically download it from Docker Hub on startup.  To avoid
echo saving this image, cancel and rerun the script with SKIP_ISVCS_DUMP=1. 
echo 
echo If you want to save any other images on your system, please cancel, then do so
echo using the "docker save" command.
echo 
echo If Control Center is not installed and you want to run this script, cancel 
echo and rerun it with NO_SERVICED=1.
echo
echo WARNING: If you are running this on a single-host system \(i.e., with 
echo SERVICED_REGISTRY=0\), this WILL destroy your current state. Cancel and save 
echo all images using the "docker save" command before continuing.
echo
confirm || exit 1


# Get the device being used for /var/lib/docker
DOCKER_DISK=$(echo "${MOUNT_LINE}" | awk {'print $1'})
log /var/lib/docker is using ${DOCKER_DISK}

ISVCS_DUMP_FILE="/tmp/isvcs-$(head -c 10 /dev/urandom | md5sum | awk {'print $1'}).tgz"
if [[ ${NO_SERVICED} -ne 1 ]] && [[ ${SKIP_ISVCS_DUMP} -ne 1 ]]; then
    # Get an isvcs image
    log Preparing to extract serviced internal services image
    ISVCS_IMAGE="$(serviced version | grep IsvcsImage | awk {'print $2'})"
    [ -z "${ISVCS_IMAGE}" ] && fail "Unable to determine which image to save. Is serviced installed properly?"
    log "Services image used by Control Center is ${ISVCS_IMAGE}"

    # Make sure Docker is running
    docker ps &>/dev/null
    if [[ $? -ne 0 ]]; then
        log Starting Docker
        ${CTL_CMD} start docker || fail Unable to start Docker
    fi
    # See if the image even exists
    if [ -n "$(docker images -q ${ISVCS_IMAGE})" ]; then
        # Dump the image
        log "Saving the internal services image"
        docker save ${ISVCS_IMAGE} | gzip -9 > ${ISVCS_DUMP_FILE}
        if [[ ${PIPESTATUS[0]} -ne 0 ]]; then
            fail "Unable to save isvcs image. To ignore, rerun this script with SKIP_ISVCS_DUMP=1"
        fi
    else
        log Skipping internal services image save as it is not present
    fi
fi

if [[ ${NO_SERVICED} -ne 1 ]]; then
    # Ensure Control Center is shut down
    log Stopping Control Center
    ${CTL_CMD} stop serviced || fail Unable to stop Control Center
fi

# Ensure Docker is shut down
log Stopping Docker
${CTL_CMD} stop docker || fail Unable to stop Docker

# Unmount /var/lib/docker
log Unmounting /var/lib/docker
umount /var/lib/docker

# Reformat the filesystem
log Reformatting ${DOCKER_DISK} with an xfs filesystem
mkfs -t xfs -f ${DOCKER_DISK}

# Modify fstab
log Modifying /etc/fstab
sed -i -e '\|/var/lib/docker| s|^|#|' /etc/fstab
echo "${DOCKER_DISK} /var/lib/docker xfs defaults 1 2" >> /etc/fstab

# Remount
mount /var/lib/docker

# Modify Docker storage driver
log Modifying Docker storage driver
if [ -f /etc/default/docker ]; then
    # Ubuntu
    DOCKER_CONFIG=/etc/default/docker
else
    # RHEL
    DOCKER_CONFIG=/etc/sysconfig/docker
fi
sed -i -e '/DOCKER_OPTS/ s/btrfs/devicemapper/' ${DOCKER_CONFIG}

# Start up Docker
log Starting Docker
${CTL_CMD} start docker || fail Unable to start Docker

# Reload the image, if it exists
if [[ ${NO_SERVICED} -ne 1 ]] && [[ ${SKIP_ISVCS_DUMP} -ne 1 ]] && [ -f "${ISVCS_DUMP_FILE}" ]; then
    log Loading internal services image
    cat "${ISVCS_DUMP_FILE}" | gunzip - | docker load || fail Unable to load internal services image
fi

log Converted Docker to use devicemapper storage backend. 
if [[ ${NO_SERVICED} -ne 1 ]]; then
    log Please start Control Center at your leisure.
fi
