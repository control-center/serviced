#!/bin/sh
#
# This script encapsulates the logic to link the local './node_modules' to a
#	predefined NPM cache.
#

# If we have a soft-link, make sure it links to an actual directory.
if [ -h ./node_modules ]; then
	LINK_TARGET=`readlink ./node_modules`
	if [ ! -d ${LINK_TARGET} ]
	then
		echo "ERROR: target=${LINK_TARGET} for soft link ./node_modules does not exist"
		exit 1
	fi
fi

if [ ! -d ./node_modules -a ! -h ./node_modules ]; then
	if [ -d /npm-cache/serviced/node_modules ]; then
		# This case should only happen when running inside the serviced-build
		#     docker container where the root user populates the cache, but
		#	  build user needs to link to it.
		NODE_MODULES_DIR=/npm-cache/serviced/node_modules
		echo "Making node_modules link to $NODE_MODULES_DIR"
		ln -s ${NODE_MODULES_DIR} ./node_modules
	fi
fi
