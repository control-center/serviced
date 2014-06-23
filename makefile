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
IN_DOCKER    = 0
NSINITDIR=../../dotcloud/docker/pkg/libcontainer/nsinit

default: build_binary

install: build_binary bash-complete
	cd web && make build_js
	cp isvcs/resources/logstash/logstash.conf.in isvcs/resources/logstash/logstash.conf
	go install
	go install github.com/dotcloud/docker/pkg/libcontainer/nsinit

bash-complete:
	sudo cp ./serviced-bash-completion.sh /etc/bash_completion.d/serviced

build_isvcs:
	cd isvcs && make IN_DOCKER=$(IN_DOCKER)

.PHONY: build_js
build_js:
	cd web && make build_js

#---------------------------------------------------------------------#
# Fail early if GOPATH not set.  Not all targets require this, but
# the bldenv is probably wonky at any rate if it is /not/ set.
#---------------------------------------------------------------------#
ifeq "$(GOPATH)" ""
    $(error "GOPATH not set.")
endif

GODEP = $(GOPATH)/bin/godep
$(GODEP): | $(GOPATH)/src/$(godep_SRC)
	go install $(godep_SRC)

godep_SRC = github.com/tools/godep
$(GOPATH)/src/$(godep_SRC):
	go get $(godep_SRC)

serviced: | $(GODEP)
	$(GODEP) restore
	go build

NSINIT = $(GOPATH)/bin/nsinit
$(NSINIT): | $(GOPATH)/src/$(nsinit_SRC)
	go install $(nsinit_SRC)

nsinit_SRC = $(docker_SRC)/pkg/libcontainer/nsinit
docker_SRC = github.com/dotcloud/docker
nsinit: | $(GOPATH)/src/$(nsinit_SRC)
	go build $($@_SRC)

.PHONY: build_binary 
build_binary: build_isvcs build_js serviced nsinit

go:
	rm serviced -Rf # temp workaround for moving main package
	go build

pkgs:
	cd pkg && $(MAKE) IN_DOCKER=$(IN_DOCKER) deb rpm

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
	cd isvcs && make export
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/bash -c "cd /go/src/github.com/zenoss/serviced/pkg/ && make clean && mkdir -p /go/src/github.com/zenoss/serviced/pkg/build/tmp"
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	-v `pwd`/pkg/build/tmp:/tmp \
	-t zenoss/serviced-build make \
		IN_DOCKER=1 \
		BUILD_NUMBER=$(BUILD_NUMBER) \
		RELEASE_PHASE=$(RELEASE_PHASE) \
		SUBPRODUCT=$(SUBPRODUCT) \
		build_binary pkgs

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

.PHONY: clean_js
clean_js:
	cd web && make clean

.PHONY: clean_nsinit
clean_nsinit:
	if [ -f nsinit ];then \
		rm nsinit ;\
	fi
	if [ -d "$(NSINITDIR)" ];then \
		cd $(NSINITDIR) && go clean ;\
	fi

.PHONY: clean_serviced
clean_serviced:
	if [ -f "serviced" ];then \
		rm serviced ;\
	fi

.PHONY: clean
clean: clean_js clean_nsinit | $(GODEP)
	rm serviced -Rf # needed for branch build to work to merge this commit, remove me later
	cd dao && make clean
	$(GODEP) restore && go clean -r && go clean -i github.com/zenoss/serviced/... # this cleans all dependencies
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/sh -c "cd /go/src/github.com/zenoss/serviced && make clean_fs" || exit 0
	go clean


clean_fs:
	cd pkg && make clean

