#!/bin/bash

source /usr/local/rvm/scripts/rvm
CUCUMBER_CMD="cd /capybara; cucumber $*"

#
# Run cucumber with the default profile (which generates a JSON report) and convert the JSON report to
# a nice HTML format
#
eval ${CUCUMBER_CMD}
EXIT=$?

if [ -d ./output ] && [ -f ./output/report.json ]; then
    java -jar /usr/share/reporter/reporter.jar ./output ./output/report.json
    chown -R $CALLER_UID:$CALLER_GID ./output
fi
exit ${EXIT}
