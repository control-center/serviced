################################################################################
#
# Copyright (C) Zenoss, Inc. 2013-2014, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################

#---------------------#
# Macros              #
#---------------------#
build_TARGETS   = build_isvcs build_js serviced nsinit
install_TARGETS = $(bash_completion) $(nsinit) $(logstash.conf) $(serviced)

# Define GOPATH for containerized builds.
#
#    NB: Keep this in sync with build/Dockerfile: ENV GOPATH /go
#
docker_GOPATH = /go

ifeq "$(GOPATH)" ""
    $(warning "GOPATH not set. Ok to ignore for containerized builds.")
else
    GOSRC = $(GOPATH)/src
    GOBIN = $(GOPATH)/bin
    GOPKG = $(GOPATH)/pkg
endif

# Avoid the inception problem of building from a container within a container.
IN_DOCKER = 0

#------------------------------------------------------------------------------#
# Build Repeatability with Godeps
#------------------------------------------------------------------------------#
# We manage go dependencies by 'godep restoring' from a checked-in list of go 
# packages at desired versions in:
#
#    ./Godeps
#
# This file is manually updated and thus requires some dev-vigilence if our 
# go imports change in name or version.
#
# Alternatively, one may run:
#
#    godep save -copy=false
#
# to generate the Godeps file based upon the src currently populated in 
# $GOPATH/src.  It may be useful to periodically audit the checked-in Godeps
# against the generated Godeps.
#------------------------------------------------------------------------------#
GODEP     = $(GOBIN)/godep
Godeps    = Godeps
godep_SRC = github.com/tools/godep

#---------------------#
# Build targets       #
#---------------------#
.PHONY: default build all
default build all: $(build_TARGETS)

.PHONY: build_binary 
build_binary: $(build_TARGETS)
	$(warning ":-[ Can we deprecate this poorly named target? [$@]')
	$(warning ":-[ We're building more than just one thing and we're building more than just binaries.")
	$(warning ":-] Why not just 'make all' or 'make serviced' if that is what you really want?")

.PHONY: build_isvcs
build_isvcs: | $(GODEP) $(Godeps)
	$(GODEP) restore
	cd isvcs && make IN_DOCKER=$(IN_DOCKER)

.PHONY: build_js
build_js:
	cd web && make build_js

# Download godep source to $GOPATH/src/.
$(GOSRC)/$(godep_SRC):
	go get $(godep_SRC)

.PHONY: go
go: 
	go build

docker_SRC = github.com/dotcloud/docker
nsinit_SRC = $(docker_SRC)/pkg/libcontainer/nsinit
nsinit: | $(GOSRC)/$(nsinit_SRC) $(GODEP) $(Godeps)
	$(GODEP) restore
	go build $($@_SRC)

serviced: | $(GODEP) $(Godeps)
	$(GODEP) restore
	go build

serviced_SRC            = github.com/zenoss/serviced
docker_serviced_SRC     = $(docker_GOPATH)/src/$(serviced_SRC)
docker_serviced_pkg_SRC = $(docker_serviced_SRC)/pkg

.PHONY: docker_build dockerbuild_binaryx
docker_build dockerbuild_binaryx: docker_ok
	docker build -t zenoss/serviced-build build
	docker run --rm \
	-v `pwd`:$(docker_serviced_SRC) \
	zenoss/serviced-build /bin/bash -c "cd $(docker_serviced_pkg_SRC) && make GOPATH=$(docker_GOPATH) clean"
	docker run --rm \
	-v `pwd`:$(docker_serviced_SRC) \
	-v `pwd`/pkg/build/tmp:/tmp \
	-t zenoss/serviced-build make GOPATH=$(docker_GOPATH) IN_DOCKER=1 build_binary
	cd isvcs && make isvcs_repo

#---------------------#
# Install targets     #
#---------------------#

bash_completion_SRC = serviced-bash-completion.sh
bash_completion     = /etc/bash_completion.d/serviced
#
# CM: This is a bit non-std to inline the sudo.  
#     More typical pattern is:
#
#        sudo make install
#
$(bash_completion): $(bash_completion_SRC)
	sudo cp $? $@

logstash.conf     = isvcs/resources/logstash/logstash.conf
logstash.conf_SRC = isvcs/resources/logstash/logstash.conf.in 
$(logstash.conf): $(logstash.conf_SRC)
	cp $? $@

# Make some things available through $(GOPATH)/bin/<thing>

GODEP: $(GODEP)
	echo building $@ from $^

# Make the installed godep primitive (under $GOPATH/bin/godep)
# dependent upon the directory that holds the godep source.
# If that directory is missing, then trigger the 'go get' of the
# source.
#
# This requires some make fu borrowed from:
#
#    https://lists.gnu.org/archive/html/help-gnu-utils/2007-08/msg00019.html
#
missing_godep_SRC = $(filter-out $(wildcard $(GOSRC)/$(godep_SRC)), $(GOSRC)/$(godep_SRC))
$(GODEP): | $(missing_godep_SRC)
	go install $(godep_SRC)

nsinit = $(GOBIN)/nsinit
missing_nsinit_SRC =  $(filter-out $(wildcard $(GOSRC)/$(nsinit_SRC)), $(GOSRC)/$(nsinit_SRC))
$(nsinit): | $(GOSRC)/$(nsinit_SRC)
	go install $(nsinit_SRC)

.PHONY: serviced_svcdef_compiler
serviced = $(GOBIN)/serviced
$(serviced) serviced_svcdef_compiler:
	go install

.PHONY: install
install: | $(build_TARGETS)
install: $(install_TARGETS)

#---------------------#
# Packaging targets   #
#---------------------#

.PHONY: pkgs
pkgs:
	cd pkg && $(MAKE) IN_DOCKER=$(IN_DOCKER) deb rpm

.PHONY: buildandpackage_within_container dockerbuildx
docker_buildandpackage dockerbuildx: docker_ok
	docker build -t zenoss/serviced-build build
	cd isvcs && make export
	docker run --rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/bash -c "cd $(docker_serviced_pkg_SRC) && make GOPATH=$(docker_GOPATH) clean && mkdir -p $(docker_serviced_pkg_SRC)/build/tmp"
	docker run --rm \
	-v `pwd`:$(docker_serviced_SRC) \
	-v `pwd`/pkg/build/tmp:/tmp \
	-t zenoss/serviced-build make \
		IN_DOCKER=1 \
		GOPATH=$(docker_GOPATH) \
		BUILD_NUMBER=$(BUILD_NUMBER) \
		RELEASE_PHASE=$(RELEASE_PHASE) \
		SUBPRODUCT=$(SUBPRODUCT) \
		build_binary pkgs

#---------------------#
# Test targets        #
#---------------------#

.PHONY: test
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
	cd validation && go test $(GOTEST_FLAGS)

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
		rm -f nsinit ;\
	fi
	if [ -d "$(GOSRC)/$(nsinit_SRC)" ];then \
		cd $(GOSRC)/$(nsinit_SRC) && go clean ;\
	fi

.PHONY: clean_serviced
clean_serviced:
	if [ -f "serviced" ];then \
		rm -f serviced ;\
	fi
	-go clean

.PHONY: clean_pkg
clean_pkg:
	cd pkg && make clean

.PHONY: clean_godeps
clean_godeps: | $(GODEP) $(Godeps)
	$(GODEP) restore && go clean -r && go clean -i github.com/zenoss/serviced/... # this cleans all dependencies

.PHONY: clean_dao
clean_dao:
	cd dao && make clean

.PHONY: clean
clean: clean_js clean_nsinit clean_pkg clean_dao clean_godeps clean_serviced

.PHONY: docker_clean_pkg
docker_clean_pkg:
	docker run --rm \
	-v `pwd`:$(docker_serviced_SRC) \
	zenoss/serviced-build /bin/bash -c "cd $(docker_serviced_pkg_SRC) && make GOPATH=$(docker_GOPATH) clean"

.PHONY: docker_clean
docker_clean: docker_clean_pkg

.PHONY: mrclean
mrclean: docker_clean clean

#==============================================================================#
# DEPRECATED STUFF -- DELETE ME SOON, PLEASE --
#==============================================================================#
dockerbuild dockerbuild_binary:
	$(error The $@ target has been deprecated. Yo, fix your makefile.)
#==============================================================================#
