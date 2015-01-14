#!/bin/sh

if [ "$#" -ne 3 ]
then
	echo "USAGE: $0 sourceRepoClass sourceVersion targetVersion"
	echo ""
	echo "Example - reversion zenoss-unstable 1.0.0_2113 to 1.0.0rc1:"
	echo "$0 zenoss-unstable 1.0.0_2113 1.0.0rc1"
	echo ""
	echo "Example - reversion testing 1.0.0rc1 to 1.0.0:"
	echo "$0 zenoss-testing 1.0.0rc1 1.0.0"
	exit 1
fi

SOURCE_CLASS=$1
SOURCE_VERSION=$2
TARGET_VERSION=$3

set -e
set -x

cd /tmp
yum install --downloadonly --downloaddir=/tmp --enablerepo=${SOURCE_CLASS} serviced-${SOURCE_VERSION}

ESCAPED_TARGET=`echo "$TARGET_VERSION" | sed 's/\./\\./g'`
SED_CMD="s/^Version:.*/Version:${ESCAPED_TARGET}/"
echo -E "$SED_CMD" >sed.cmd

SOURCE_RPM=serviced-${SOURCE_VERSION}-1.x86_64.rpm
TARGET_RPM=serviced-${TARGET_VERSION}-1.x86_64.rpm

echo -e "\nFYI - Here's the metadata for the source package"
rpm -qip ${SOURCE_RPM}

rpmrebuild --notest-install --directory=/tmp --change-spec-preamble='sed -f sed.cmd' --package ./${SOURCE_RPM}

echo -e "\nFYI - Here's the metadata for the target package"
rpm -qip x86_64/${TARGET_RPM}

mv x86_64/${TARGET_RPM} /output
