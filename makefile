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
	go get github.com/zenoss/glog
	go build && go test
	cd client && make
	cd svc && make 
	cd agent && make
	cd web && make
	cd proxy && make
	cd dao && make
	cd serviced && make


clean:
	go clean
	cd client && make clean
	cd serviced && make clean
	cd agent && make clean
	cd svc && make clean
	cd proxy && make clean
	cd dao && make clean
