################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################

default: elastigo
	@echo "Executing make style build. You can also use the 'go' tool."
	go get github.com/coopernurse/gorp
	go get github.com/ziutek/mymysql/godrv
	go get github.com/zenoss/glog
	go get github.com/samuel/go-zookeeper/zk
	go get github.com/araddon/gou
	go build && go test
	cd client && make
	cd svc && make 
	cd agent && make
	cd web && make
	cd proxy && make
	cd dao && make
	cd serviced && make


elastigo:../../mattbaird/elastigo

../../mattbaird/elastigo:
	mkdir ../../mattbaird -p && \
	cd ../../mattbaird && \
	git clone git@github.com:zenoss/elastigo.git

clean:
	go clean
	cd client && make clean
	cd serviced && make clean
	cd agent && make clean
	cd svc && make clean
	cd proxy && make clean
	cd dao && make clean
