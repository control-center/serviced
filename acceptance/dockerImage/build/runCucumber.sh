#!/bin/bash

#
# Make sure that Xvfb is started in case we're using Chrome or Firefox
# (Xvfb is not used by poltergeist/phantomjs)
if [ "${CAPYBARA_DRIVER}" != "poltergeist" ]; then
    /etc/init.d/xvfb start
fi

source /usr/local/rvm/scripts/rvm
CUCUMBER_CMD="cd /capybara; cucumber $*"

#
# Run cucumber with the default profile (which generates a JSON report) and convert the JSON report to
# a nice HTML format
#
eval ${CUCUMBER_CMD}
EXIT=$?

java -jar /usr/share/reporter/reporter.jar ./output ./output/report.json
chown -R $CALLER_UID:$CALLER_GID ./output
exit $EXIT
