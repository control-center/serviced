################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################

default:
	@echo "Executing make style build. You can also use the 'go' tool."
	go get github.com/coopernurse/gorp
	go get github.com/ziutek/mymysql/godrv
	go build && go test
	cd serviced && go build && go test

clean:
	go clean
	cd serviced && make clean


