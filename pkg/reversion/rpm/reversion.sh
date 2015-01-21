#!/bin/bash

if [ "$#" -ne 5 ]
then
        echo "USAGE: $0 sourceRepoClass sourceVersion sourceRelease targetVersion targetRelease"
        echo ""
        echo "Example - reversion zenoss-unstable 1.0.0-0.0.2113.unstable to 1.0.0-0.1.CR1:"
        echo "$0 zenoss-unstable 1.0.0 0.0.2113.unstable 1.0.0 0.1.CR1"
        echo ""
        echo "Example - reversion testing 1.0.0-0.1.CR1 to 1.0.0-1:"
        echo "$0 zenoss-testing 1.0.0 0.1.RC1 1.0.0 1"
        exit 1
fi

SOURCE_CLASS="$1"
SOURCE_VERSION="$2"
SOURCE_RELEASE="$3"
TARGET_VERSION="$4"
TARGET_RELEASE="$5"

set -e
set -x

cd /tmp
yum install --downloadonly --downloaddir=/tmp --enablerepo=${SOURCE_CLASS} serviced-${SOURCE_VERSION}-${SOURCE_RELEASE}

ESCAPED_VERSION=`echo "$TARGET_VERSION" | sed 's/\./\\./g'`
VERSION_SED_CMD="s/^Version:.*/Version:${ESCAPED_VERSION}/"
echo -E "$VERSION_SED_CMD" > sed.cmd

SOURCE_RPM=serviced-${SOURCE_VERSION}-${SOURCE_RELEASE}.x86_64.rpm
TARGET_RPM=serviced-${TARGET_VERSION}-${TARGET_RELEASE}.x86_64.rpm

echo -e "\nFYI - Here's the metadata for the source package"
rpm -qip ${SOURCE_RPM}

rpmrebuild --release="$TARGET_RELEASE" --notest-install --directory=/tmp --change-spec-preamble="sed -f sed.cmd"  --package ./${SOURCE_RPM}

echo -e "\nFYI - Here's the metadata for the target package"
rpm -qip x86_64/${TARGET_RPM}

mv x86_64/${TARGET_RPM} /output/
