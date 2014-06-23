################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################
build_TARGETS = build_isvcs build_js serviced nsinit
default: $(build_TARGETS)

#---------------------#
# Build macros        #
#---------------------#

docker_SRC = github.com/dotcloud/docker
#
# Fail early if GOPATH not set.  Not all targets require 
# this, but the bldenv is probably wonky otherwise.
#
ifeq "$(GOPATH)" ""
    $(error "GOPATH not set.")
else
    GOSRC = $(GOPATH)/src
    GOBIN = $(GOPATH)/bin
    GOPKG = $(GOPATH)/pkg
endif
GODEP      = $(GOBIN)/godep
IN_DOCKER  = 0
nsinit_SRC = $(docker_SRC)/pkg/libcontainer/nsinit
NSINIT     = $(GOBIN)/nsinit
SERVICED   = $(GOBIN)/serviced

#---------------------#
# Build targets       #
#---------------------#

.PHONY: build_binary 
build_binary: $(build_TARGETS)

.PHONY: build_js
build_js:
	cd web && make build_js

.PHONY: build_isvcs
build_isvcs: | $(GODEP)
	$(GODEP) restore
	cd isvcs && make IN_DOCKER=$(IN_DOCKER)

# Download godep source to $GOPATH/src/.
godep_SRC = github.com/tools/godep
$(GOSRC)/$(godep_SRC):
	go get $(godep_SRC)

.PHONY: go
go:
	go build

nsinit: | $(GOSRC)/$(nsinit_SRC)
	go build $($@_SRC)

.PHONY: serviced_svcdef_compiler
serviced: | $(GODEP)
	$(GODEP) restore
	go build

.PHONY: build_within_container dockerbuild_binaryx
build_within_container dockerbuild_binaryx: docker_ok
	docker build -t zenoss/serviced-build build
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/bash -c "cd /go/src/github.com/zenoss/serviced/pkg/ && make clean && mkdir -p /go/src/github.com/zenoss/serviced/pkg/build/tmp"
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	-v `pwd`/pkg/build/tmp:/tmp \
	-t zenoss/serviced-build make IN_DOCKER=1 build_binary
	cd isvcs && make isvcs_repo

#---------------------#
# Install targets     #
#---------------------#

.PHONY: bash-complete
bash-complete:
	sudo cp ./serviced-bash-completion.sh /etc/bash_completion.d/serviced

install_logstash.conf = isvcs/resources/logstash/logstash.conf
logstash.conf_SRC     = isvcs/resources/logstash/logstash.conf.in 
$(install_logstash.conf): $(logstash.conf_SRC)
	cp $? $@

# Make some things available through $(GOPATH)/bin/thing

$(GODEP): | $(GOSRC)/$(godep_SRC)
	go install $(godep_SRC)

$(NSINIT): | $(GOSRC)/$(nsinit_SRC)
	go install $(nsinit_SRC)

$(SERVICED) serviced_svcdef_compiler:
	go install .

.PHONY: install
# :-( Ug.  Build targets should not trigger install targets.
#          Should probably be install: | <build_targets...>
#
install: build_binary bash-complete build_js $(NSINIT) $(install_logstash.conf)
	go install

#---------------------#
# Packaging targets   #
#---------------------#

.PHONY: pkgs
pkgs:
	cd pkg && $(MAKE) IN_DOCKER=$(IN_DOCKER) deb rpm

.PHONY: buildandpackage_within_container dockerbuildx
buildandpackage_within_container dockerbuildx: docker_ok
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

#---------------------#
# Test targets        #
#---------------------#

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

#---------------------#
# Clean targets       #
#---------------------#

.PHONY: clean_js
clean_js:
	cd web && make clean

.PHONY: clean_nsinit
clean_nsinit:
	if [ -f nsinit ];then \
		rm nsinit ;\
	fi
	if [ -d "$(GOSRC)/$(nsinit_SRC)" ];then \
		cd $(GOSRC)/$(nsinit_SRC) && go clean ;\
	fi

.PHONY: clean_serviced
clean_serviced:
	if [ -f "serviced" ];then \
		rm serviced ;\
	fi

# Better name for this target, please.
# Should this be chained to the clean target?
.PHONY: clean_fs
clean_fs:
	cd pkg && make clean

# This needs more work.
.PHONY: clean
clean: clean_js clean_nsinit | $(GODEP)
	rm serviced -Rf # needed for branch build to work to merge this commit, remove me later
	cd dao && make clean
	$(GODEP) restore && go clean -r && go clean -i github.com/zenoss/serviced/... # this cleans all dependencies
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/sh -c "cd /go/src/github.com/zenoss/serviced && make clean_fs" || exit 0
	go clean


#==============================================================================#
# DEPRECATED STUFF -- DELETE ME SOON, PLEASE --
#==============================================================================#
dockerbuild dockerbuild_binary:
	$(error The $@ target has been deprecated. Yo, fix your makefile.)

ifeq "0" "1"
pwdchecksum  := $(shell pwd | md5sum | awk '{print $$1}')
dockercache  := /tmp/serviced-dind-$(pwdchecksum)

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
endif
#==============================================================================#
