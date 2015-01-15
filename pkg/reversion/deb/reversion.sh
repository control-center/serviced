#!/bin/sh

if [ "$#" -ne 3 ]
then
	echo "ERROR: incorrect number of arguments"
	echo "USAGE: $0 sourceRepoClass sourceVersion targetVersion"
	echo ""
	echo "Example - reversion unstable 1.0.0-2113~trusty to 1.0.0rc1:"
	echo "$0 unstable 1.0.0-2113~trusty 1.0.0rc1"
	echo ""
	echo "Example - reversion testing 1.0.0rc1 to 1.0.0:"
	echo "$0 testing 1.0.0rc1 1.0.0"
	exit 1
fi

SOURCE_CLASS="$1"
SOURCE_VERSION="$2"
TARGET_VERSION="$3"

set -e
set -x

# Add the source repo
REPO_URL=http://${SOURCE_CLASS}.zenoss.io/apt/ubuntu
sh -c 'echo "deb [ arch=amd64 ] '$REPO_URL' trusty universe" \
  > /etc/apt/sources.list.d/zenoss.list'
apt-get update

cd /tmp
apt-get download serviced=${SOURCE_VERSION}

echo -e "\nFYI - Here's the metadata for the source package"
dpkg -f serviced_${SOURCE_VERSION}_amd64.deb

deb-reversion -b -v ${TARGET_VERSION} serviced_${SOURCE_VERSION}_amd64.deb

echo -e "\nFYI - Here's the metadata for the target package"
dpkg -f serviced_${TARGET_VERSION}_amd64.deb

mv serviced_${TARGET_VERSION}_amd64.deb /output
