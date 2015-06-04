#!/bin/bash

#
# Make sure that Xvfb is started in case we're using Chrome or Firefox
# (Xvfb is not used by poltergeist/phantomjs)
if [ "${CAPYBARA_DRIVER}" != "poltergeist" ]; then
    /etc/init.d/xvfb start
fi

runAsRoot=false
if [ $# -gt 0 -a "$1" == "--root" ]; then
	shift 1
	runAsRoot=true
fi

CUCUMBER_CMD="cd /capybara; cucumber $*; java -jar /usr/share/reporter/reporter.jar ./output ./output/report.json"
if [ "$runAsRoot" == "true" ]; then
	eval ${CUCUMBER_CMD}
else
	#
	# Make the user account 'cuke' in the container which matches the UID/GID of
	# the caller outside of the container
	#
	makeCukeUser.sh

	#
	# Run cucumber with the default profile (which generates a JSON report) and convert the JSON report to
	# a nice HTML format
	#
	HOME=/home/cuke su cuke --preserve-environment -c "${CUCUMBER_CMD}"
fi
