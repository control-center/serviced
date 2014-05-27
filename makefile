################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################

pwdchecksum := $(shell pwd | md5sum | awk '{print $$1}')
dockercache := /tmp/serviced-dind-$(pwdchecksum)
IN_DOCKER := 0

default: build_binary

install: build_binary bash-complete
	cd web && make build-js
	cp isvcs/resources/logstash/logstash.conf.in isvcs/resources/logstash/logstash.conf
	go install github.com/zenoss/serviced/serviced
	go install github.com/dotcloud/docker/pkg/libcontainer/nsinit/nsinit

bash-complete:
	sudo cp ./serviced/serviced-bash-completion.sh /etc/bash_completion.d/serviced

build_binary:
	cd serviced && make
	if [ "$(IN_DOCKER)" = "0" ]; then \
		cd isvcs && make; \
	else \
		cd isvcs && make buildgo; \
	fi

go:
	cd serviced && go build

pkgs:
	cd pkg && make IN_DOCKER=$(IN_DOCKER) rpm && make IN_DOCKER=$(IN_DOCKER) deb

dockerbuild_binaryx: docker_ok
	docker build -t zenoss/serviced-build build
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/bash -c "cd /go/src/github.com/zenoss/serviced/pkg/ && make clean && mkdir -p /go/src/github.com/zenoss/serviced/pkg/build/tmp"
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	-v `pwd`/pkg/build/tmp:/tmp \
	-e BUILD_NUMBER=$(BUILD_NUMBER) -t \
	zenoss/serviced-build make IN_DOCKER=1 build_binary
	serviced/godep restore
	cd isvcs && make isvcs_repo

dockerbuild_binary: docker_ok
	docker build -t zenoss/serviced-build build
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/bash -c "cd /go/src/github.com/zenoss/serviced/pkg/ && make clean && mkdir -p /go/src/github.com/zenoss/serviced/pkg/build/tmp"
	echo "Using dock-in-docker cache dir $(dockercache)"
	mkdir -p $(dockercache)
	time docker run --rm \
	--privileged \
	-v $(dockercache):/var/lib/docker \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	-v `pwd`/pkg/build/tmp:/tmp \
	-e BUILD_NUMBER=$(BUILD_NUMBER) -t \
	zenoss/serviced-build /bin/bash \
	-c '/usr/local/bin/wrapdocker && make build_binary'

dockerbuildx: docker_ok
	docker build -t zenoss/serviced-build build
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/bash -c "cd /go/src/github.com/zenoss/serviced/pkg/ && make clean && mkdir -p /go/src/github.com/zenoss/serviced/pkg/build/tmp"
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	-v `pwd`/pkg/build/tmp:/tmp \
	-e BUILD_NUMBER=$(BUILD_NUMBER) -t \
	zenoss/serviced-build make IN_DOCKER=1 build_binary pkgs

dockerbuild: docker_ok
	docker build -t zenoss/serviced-build build
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/bash -c "cd /go/src/github.com/zenoss/serviced/pkg/ && make clean && mkdir -p /go/src/github.com/zenoss/serviced/pkg/build/tmp"
	echo "Using dock-in-docker cache dir $(dockercache)"
	mkdir -p $(dockercache)
	time docker run --rm \
	--privileged \
	-v $(dockercache):/var/lib/docker \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	-v `pwd`/pkg/build/tmp:/tmp \
	-e BUILD_NUMBER=$(BUILD_NUMBER) -t \
	zenoss/serviced-build /bin/bash \
	-c '/usr/local/bin/wrapdocker && make build_binary pkgs'

test: build_binary docker_ok
	go test ./commons/... $(GOTEST_FLAGS)
	go test $(GOTEST_FLAGS)
	cd dao && make test
	cd web && go test $(GOTEST_FLAGS)
	cd serviced && go test $(GOTEST_FLAGS)
	cd utils && go test $(GOTEST_FLAGS)
	cd datastore && make test $(GOTEST_FLAGS)
	cd domain && make test $(GOTEST_FLAGS)
	cd facade && go test $(GOTEST_FLAGS)
	cd rpc && make test $(GOTEST_FLAGS)
	cd cli/api && go test $(GOTEST_FLAGS)
	cd cli/cmd && go test $(GOTEST_FLAGS)
	cd scheduler && go test $(GOTEST_FLAGS)
	cd container && go test $(GOTEST_FLAGS)
	cd dfs/nfs && go test $(GOTEST_FLAGS)
	cd coordinator/client && go test $(GOTEST_FLAGS)
	cd coordinator/storage && go test $(GOTEST_FLAGS)

smoketest: build_binary docker_ok
	/bin/bash smoke.sh

docker_ok:
	if docker ps >/dev/null; then \
		echo "docker OK"; \
	else \
		echo "Check 'docker ps' command"; \
		exit 1;\
	fi

clean:
	cd dao && make clean
	cd serviced && ./godep restore && go clean -r && go clean -i github.com/zenoss/serviced/... # this cleans all dependencies
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/sh -c "cd /go/src/github.com/zenoss/serviced && make clean_fs" || exit 0

clean_fs:
	cd pkg && make clean

