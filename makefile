# Copyright 2014 The Serviced Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

VERSION := $(shell cat ./VERSION)
DATE := '$(shell date -u)'

# GIT_URL ?= $(shell git remote show origin | grep 'Fetch URL' | awk '{ print $$3 }')
# assume it will get set because the above can cause network traffic on every run
GIT_COMMIT ?= $(shell ./gitstatus.sh)
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)

GOBUILD_TAGS  ?= $(shell bash build-tags.sh)
GOBUILD_FLAGS ?= -tags "$(GOBUILD_TAGS)"



# jenkins default, jenkins-${JOB_NAME}-${BUILD_NUMBER}
BUILD_TAG ?= 0


LDFLAGS = -ldflags "-X main.Version $(VERSION) -X main.Giturl '$(GIT_URL)' -X main.Gitcommit $(GIT_COMMIT) -X main.Gitbranch $(GIT_BRANCH) -X main.Date $(DATE) -X main.Buildtag $(BUILD_TAG)"

#---------------------#
# Macros              #
#---------------------#

install_TARGETS   = $(install_DIRS)
prefix            = /opt/serviced
sysconfdir        = /etc

#
# Specify if we want service definition templates (picked up from
# pkg/templates) to be included as part of the serviced packaging.
#
INSTALL_TEMPLATES      ?= 1

#
# When packaging just the templates, throw this option.
#
INSTALL_TEMPLATES_ONLY = 0

# The installed footprint is influenced by the distro
# we're targeting.  Allow this usage:
#
#    sudo make install PKG=<deb|rpm>
#
PKG         = $(default_PKG) # deb | rpm | tgz
default_PKG = deb

build_TARGETS = build_isvcs build_js serviced serviced-controller

# Define GOPATH for containerized builds.
#
#    NB: Keep this in sync with build/Dockerfile: ENV GOPATH /go
#
docker_GOPATH = /go

serviced_SRC            = github.com/control-center/serviced
docker_serviced_SRC     = $(docker_GOPATH)/src/$(serviced_SRC)
docker_serviced_pkg_SRC = $(docker_serviced_SRC)/pkg

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
# We manage go dependencies by 'godep saving' from the current $GOPATH/src.
# The Godeps directory is manually updated and thus requires some dev-vigilence
# if our go imports change in name or version.
#
#    godep save ./...
#
# to generate the Godeps file based upon the src currently populated in
# $GOPATH/src.  It may be useful to periodically audit the checked-in Godeps
# against the generated Godeps.
#------------------------------------------------------------------------------#
GODEP     = $(GOBIN)/godep
GO        = $(GODEP) go
godep_SRC = github.com/tools/godep

# Normalize DESTDIR so we can use this idiom in our install targets:
#
# $(_DESTDIR)$(prefix)
#
# and not end up with double slashes.
ifneq "$(DESTDIR)" ""
    PREFIX_HAS_LEADING_SLASH = $(patsubst /%,/,$(prefix))
    ifeq "$(PREFIX_HAS_LEADING_SLASH)" "/"
        _DESTDIR := $(shell echo $(DESTDIR) | sed -e "s|\/$$||g")
    else
        _DESTDIR := $(shell echo $(DESTDIR) | sed -e "s|\/$$||g" -e "s|$$|\/|g")
    endif
endif

#---------------------#
# Build targets       #
#---------------------#
.PHONY: default build all
default build all: $(build_TARGETS)

.PHONY: build_isvcs
build_isvcs:
	cd isvcs && make

.PHONY: build_js
build_js:
	cd web/ui && make clean build

.PHONY: mockAgent
mockAgent:
	cd acceptance/mockAgent && $(GO) build $(GOBUILD_FLAGS) ${LDFLAGS}

# Download godep source to $GOPATH/src/.
$(GOSRC)/$(godep_SRC):
	go get $(godep_SRC)

GOVET     = $(GOBIN)/govet
govet_SRC = golang.org/x/tools/cmd/vet

# Download govet source to $GOPATH/src/.
$(GOSRC)/$(govet_SRC):
	go get $(govet_SRC)

#
# FIXME: drop -composites=false to get full coverage
GOVET_EXCLUDE_DIRS = Godeps/ build/ chef/ vagrant/
GOVET_TARGET_DIRS =  $(filter-out $(GOVET_EXCLUDE_DIRS), $(sort $(dir $(wildcard */*))))
govet: $(GOSRC)/$(govet_SRC)
	@echo "GOVET_TARGET_DIRS='${GOVET_TARGET_DIRS}'"
	go tool vet -composites=false $(GOVET_FLAGS) $(GOVET_TARGET_DIRS)

.PHONY: go
go:
	$(GO) build $(GOBUILD_FLAGS) ${LDFLAGS}
	cd serviced-controller && $(GO) build $(GOBUILD_FLAGS) ${LDFLAGS}

# As a dev convenience, we call both 'go build' and 'go install'
# so the current directory and $GOPATH/bin are updated
# with the built target.  This allows dev's to reference the target out
# of their GOPATH and type <goprog> instead of the laborious ./<goprog> :-)

docker_SRC = github.com/docker/docker


# https://www.gnu.org/software/make/manual/html_node/Force-Targets.html
#
# Force our go recipies to always fire since make doesn't
# understand all of the target's *.go dependencies.  In this case let
# '$(GO) build' determine if the target needs to be rebuilt.
FORCE:

serviced: $(GODEP)
serviced: FORCE
	$(GO) build $(GOBUILD_FLAGS) ${LDFLAGS}
	make govet
	if [ -n "$(GOBIN)" ]; then cp serviced $(GOBIN)/serviced; fi

serviced-controller: $(GODEP)
serviced-controller: FORCE
	cd serviced-controller && $(GO) build $(GOBUILD_FLAGS) ${LDFLAGS}
	if [ -n "$(GOBIN)" ]; then cp serviced-controller/serviced-controller $(GOBIN)/serviced-controller; fi
#
# BUILD_VERSION is the version of the serviced-build docker image
#
BUILD_VERSION = v$(VERSION)-1
export BUILD_VERSION

#
# This target is used to rebuild the zenoss/serviced-build image.
# It is intended to be run manually whenever the contents of the build image change.
# It should not be run as part for the regular local build process, or the
# regular Jenkins build process.
#
buildServicedBuildImage: docker_ok buildServicedBuildImage_ok
	cp web/ui/package.json build
	cp web/ui/npm-shrinkwrap.json build
	docker build -t zenoss/serviced-build:$(BUILD_VERSION) build

#
# This target is used to push the zenoss/serviced-build image to docker hub.
# It is intended to be run manually whenever the contents of the build image change.
# It should not be run as part for the regular local build process, or the
# regular Jenkins build process.
#
pushServicedBuildImage: docker_ok
	docker push zenoss/serviced-build:$(BUILD_VERSION)

buildServicedBuildImage_ok:
ifndef forceDockerBuild
	@echo "Checking to see if zenoss/serviced-build:$(BUILD_VERSION) exists"
	@if docker images | grep zenoss/serviced-build | grep $(BUILD_VERSION) >/dev/null; then \
		echo "ERROR: zenoss/serviced-build:$(BUILD_VERSION) already exists"; \
		echo "       To replace it, use"; \
		echo "       make forceDockerBuild=true buildServicedBuildImage"; \
		exit 1; \
	else \
		echo "Verified zenoss/serviced-build:$(BUILD_VERSION) does NOT exist"; \
	fi
else
	@echo "Skipping check to see if zenoss/serviced-build:$(BUILD_VERSION) exists"
	@echo "Replacing zenoss/serviced-build:$(BUILD_VERSION) (if it already exists)";
endif

.PHONY: docker_build
pkg_build_tmp = pkg/build/tmp
docker_build: docker_ok
	docker run --rm \
	-v `pwd`:$(docker_serviced_SRC) \
	zenoss/serviced-build:$(BUILD_VERSION) /bin/bash -c "cd $(docker_serviced_pkg_SRC) && make GOPATH=$(docker_GOPATH) clean"
	if [ ! -d "$(pkg_build_tmp)" ];then \
		mkdir -p $(pkg_build_tmp) ;\
	fi
	docker run --rm \
	-v `pwd`:$(docker_serviced_SRC) \
	-v `pwd`/$(pkg_build_tmp):/tmp \
	-t zenoss/serviced-build:$(BUILD_VERSION) \
	make GOPATH=$(docker_GOPATH) IN_DOCKER=1 build

# Make the installed godep primitive (under $GOPATH/bin/godep)
# dependent upon the directory that holds the godep source.
# If that directory is missing, then trigger the '$(GO) install' of the
# source.
#
# This requires some make fu borrowed from:
#
#    https://lists.gnu.org/archive/html/help-gnu-utils/2007-08/msg00019.html
#
missing_godep_SRC = $(filter-out $(wildcard $(GOSRC)/$(godep_SRC)), $(GOSRC)/$(godep_SRC))
$(GODEP): | $(missing_godep_SRC)
	go install $(godep_SRC)

#---------------------#
# Install targets     #
#---------------------#

install_DIRS  = $(_DESTDIR)$(prefix)
install_DIRS += $(_DESTDIR)/usr/bin
install_DIRS += $(_DESTDIR)$(prefix)/bin
install_DIRS += $(_DESTDIR)$(prefix)/etc
install_DIRS += $(_DESTDIR)$(prefix)/doc
install_DIRS += $(_DESTDIR)$(prefix)/share/web
install_DIRS += $(_DESTDIR)$(prefix)/share/shell
install_DIRS += $(_DESTDIR)$(prefix)/isvcs
install_DIRS += $(_DESTDIR)$(sysconfdir)/default
install_DIRS += $(_DESTDIR)$(sysconfdir)/bash_completion.d
install_DIRS += $(_DESTDIR)$(sysconfdir)/cron.daily

# Specify the stuff to install as attributes of the various
# install directories we know about.
#
# Usage:
#
#     $(dir)_TARGETS = filename
#     $(dir)_TARGETS = src_filename:dest_filename
#
default_INSTCMD = cp
$(_DESTDIR)$(sysconfdir)/cron.daily_TARGETS        = pkg/cron.daily:serviced
$(_DESTDIR)$(prefix)/etc_TARGETS                   = pkg/serviced.logrotate:logrotate.conf
$(_DESTDIR)$(prefix)/bin_TARGETS                   = serviced
$(_DESTDIR)$(prefix)/bin_TARGETS                  += serviced-controller/serviced-controller:serviced-controller
$(_DESTDIR)$(prefix)/bin_TARGETS                  += pkg/serviced-container-cleanup:serviced-container-cleanup
$(_DESTDIR)$(prefix)/bin_TARGETS                  += pkg/serviced-container-usage:serviced-container-usage
$(_DESTDIR)$(prefix)/bin_LINK_TARGETS             += $(prefix)/bin/serviced:$(_DESTDIR)/usr/bin/serviced
$(_DESTDIR)$(prefix)/share/web_TARGETS             = web/ui/build:static
$(_DESTDIR)$(prefix)/share/web_INSTOPT             = -R
$(_DESTDIR)$(prefix)/share/shell_TARGETS           = shell/static:.
$(_DESTDIR)$(prefix)/share/shell_INSTOPT           = -R
$(_DESTDIR)$(prefix)/isvcs_TARGETS                 = isvcs/resources:.
$(_DESTDIR)$(prefix)/isvcs_INSTOPT                 = -R
$(_DESTDIR)$(sysconfdir)/default_TARGETS           = pkg/serviced.default:serviced
$(_DESTDIR)$(sysconfdir)/bash_completion.d_TARGETS = serviced-bash-completion.sh:serviced

#-----------------------------------#
# Install targets (distro-specific) #
#-----------------------------------#
_PKG = $(strip $(PKG))
ifeq "$(_PKG)" "deb"
install_DIRS += $(_DESTDIR)$(sysconfdir)/init
endif
ifeq "$(_PKG)" "rpm"
install_DIRS += $(_DESTDIR)/usr/lib/systemd/system
endif

ifeq "$(_PKG)" "deb"
$(_DESTDIR)$(sysconfdir)/init_TARGETS      = pkg/serviced.upstart:serviced.conf
endif
ifeq "$(_PKG)" "rpm"
$(_DESTDIR)/usr/lib/systemd/system_TARGETS = pkg/serviced.service:serviced.service
$(_DESTDIR)$(prefix)/bin_TARGETS		  += pkg/serviced-systemd.sh:serviced-systemd.sh
endif

#-----------------------------------#
# Install targets (service defs)    #
#-----------------------------------#

# We're moving toward packaging service definitions by themselves.
# Define the policies that control when templates show up under the
# staged install directory (i.e., $(PKGROOT)/opt/serviced/templates)
# consumed at package time.
#
# TODO: Revisit where to tuck these templates relative to FHS.

# NB: If either INSTALL_TEMPLATES or INSTALL_TEMPLATES_ONLY is asserted
#     then jump into the body of the ifneq and augment install_DIRS and
#     targets accordingly.

ifneq (,$(filter 1,$(INSTALL_TEMPLATES) $(INSTALL_TEMPLATES_ONLY)))
    ifeq "$(INSTALL_TEMPLATES_ONLY)" "1"
        # Install just the service definitions in preparation
        # for creating servicedef packages.
        install_DIRS  = $(_DESTDIR)$(prefix)/templates
    else
        # Include svcdefs with serviced deb.
        install_DIRS += $(_DESTDIR)$(prefix)/templates
    endif

    # At the moment, the pkg/templates directory is actually
    # populated by our top-level makefile.  This seems a bit disjoint.
    # Will fix once I figure out some cleaner.

    $(_DESTDIR)$(prefix)/templates_TARGETS = pkg/templates/:.
    $(_DESTDIR)$(prefix)/templates_INSTCMD = rsync
    $(_DESTDIR)$(prefix)/templates_INSTOPT = -a --exclude=README.txt
endif

# Iterate across all the install dirs, populating
# same with install targets (e.g., files, directories).
#
$(install_DIRS): install_TARGETS = $($@_TARGETS)
$(install_DIRS): install_LINK_TARGETS = $($@_LINK_TARGETS)
$(install_DIRS): instcmd = $(firstword $($@_INSTCMD) $(default_INSTCMD))
$(install_DIRS): instopt = $($@_INSTOPT)
$(install_DIRS): FORCE
	@for install_DIR in $@ ;\
	do \
		if [ ! -d "$${install_DIR}" ];then \
			echo "mkdir -p $${install_DIR}" ;\
			mkdir -p $${install_DIR};\
			rc=$$? ;\
			if [ $${rc} -ne 0 ];then \
				echo "[$@] Try: 'sudo make install' or 'make install DESTDIR=/tmp/root'" ;\
				exit $${rc} ;\
			fi ;\
		fi ;\
		for install_TARGET in $(install_TARGETS) ;\
		do \
			case $${install_TARGET} in \
				*:*) \
					from=`echo $${install_TARGET} | cut -d: -f1`;\
					to=`echo $${install_TARGET} | cut -d: -f2` ;\
					;;\
				*) \
					from=$${install_TARGET} ;\
					to=$${install_TARGET} ;\
					;;\
			esac ;\
			if [ -e "$${from}" ];then \
				echo "$(instcmd) $(instopt) $${from} $${install_DIR}/$${to}" ;\
				$(instcmd) $(instopt) $${from} $${install_DIR}/$${to} ;\
				rc=$$? ;\
				if [ $${rc} -ne 0 ];then \
					exit $${rc} ;\
				fi ;\
			else \
				echo "[$@] Missing $${from}" ;\
				echo "[$@] Try: 'make build'" ;\
				exit 1 ;\
			fi ;\
		done ;\
		for install_LINK_TARGET in $(install_LINK_TARGETS) ;\
		do \
			case $${install_LINK_TARGET} in \
				*:*) \
					from=`echo $${install_LINK_TARGET} | cut -d: -f1`;\
					to=`echo $${install_LINK_TARGET} | cut -d: -f2` ;\
					;;\
				*) \
					from=$${install_LINK_TARGET} ;\
					to= ;\
					;;\
			esac ;\
			if [ -e "$(_DESTDIR)$${from}" ];then \
				echo "ln -sf $${from} $${to}" ;\
				ln -sf $${from} $${to} ;\
				rc=$$? ;\
				if [ $${rc} -ne 0 ];then \
					exit $${rc} ;\
				fi ;\
			else \
				echo "[$@] Missing $(_DESTDIR)$${from}" ;\
				echo "[$@] Try: 'make build && make install'" ;\
				exit 1 ;\
			fi ;\
		done ;\
	done

.PHONY: install
install: $(install_TARGETS)

#---------------------#
# Packaging targets   #
#---------------------#

PKGS = deb rpm tgz
.PHONY: pkgs
pkgs:
	cd pkg && $(MAKE) IN_DOCKER=$(IN_DOCKER) INSTALL_TEMPLATES=$(INSTALL_TEMPLATES) $(PKGS)

# Both BUILD_NUMBER and RELEASE_PHASE cannot be empty.
# If not providing a BUILD_NUMBER, then it is assumed that this is not a nightly build,
# and a RELEASE_PHASE must be provided (ALPHA1, BETA1, RC1, etc)

.PHONY: docker_buildandpackage
docker_buildandpackage: docker_ok
	if [ -z "$$RELEASE_PHASE" -a -z "$$BUILD_NUMBER" ]; then \
        exit 1 ;\
    fi
	docker run --rm \
	-v `pwd`:/go/src/github.com/control-center/serviced \
	zenoss/serviced-build:$(BUILD_VERSION) /bin/bash -c "cd $(docker_serviced_pkg_SRC) && make GOPATH=$(docker_GOPATH) clean"
	if [ ! -d "$(pkg_build_tmp)" ];then \
		mkdir -p $(pkg_build_tmp) ;\
	fi
	docker run --rm \
	-v `pwd`:$(docker_serviced_SRC) \
	-v `pwd`/$(pkg_build_tmp):/tmp \
	-t zenoss/serviced-build:$(BUILD_VERSION) make \
		IN_DOCKER=1 \
		INSTALL_TEMPLATES=$(INSTALL_TEMPLATES) \
		GOPATH=$(docker_GOPATH) \
		BUILD_TAG=$(BUILD_TAG) \
		BUILD_NUMBER=$(BUILD_NUMBER) \
		RELEASE_PHASE=$(RELEASE_PHASE) \
		SUBPRODUCT=$(SUBPRODUCT) \
		build pkgs

#---------------------#
# Test targets        #
#---------------------#

.PHONY: test
test: build docker_ok
	./run-tests.sh
	cd web && make "GO=$(GO)" test

smoketest: build docker_ok
	/bin/bash smoke.sh

docker_ok:
	@if docker ps >/dev/null; then \
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
	cd web/ui && make clean

.PHONY: clean_serviced
clean_serviced: $(GODEP)
	@for target in serviced $(serviced) ;\
        do \
                if [ -f "$${target}" ];then \
                        rm -f $${target} ;\
			echo "rm -f $${target}" ;\
                fi ;\
        done
	-$(GO) clean

.PHONY: clean_pkg
clean_pkg:
	cd pkg && make clean

.PHONY: clean_dao
clean_dao: $(GODEP)
	cd dao && make "GO=$(GO)" clean

.PHONY: clean
clean: clean_js clean_pkg clean_dao clean_serviced

.PHONY: docker_clean_pkg
docker_clean_pkg:
	docker run --rm \
	-v `pwd`:$(docker_serviced_SRC) \
	zenoss/serviced-build:$(BUILD_VERSION) /bin/bash -c "cd $(docker_serviced_pkg_SRC) && make GOPATH=$(docker_GOPATH) clean"

.PHONY: docker_clean
docker_clean: docker_clean_pkg

.PHONY: mrclean
mrclean: docker_clean clean

#==============================================================================#
# DEPRECATED STUFF -- DELETE ME SOON, PLEASE --
#==============================================================================#
.PHONY: dockerbuild dockerbuild_binary dockerbuildx dockerbuild_binaryx
dockerbuild dockerbuild_binary dockerbuildx dockerbuild_binaryx:
	$(error The $@ target has been deprecated. Yo, fix your makefile. Use docker_build or possibly docker_buildandpackage.)

.PHONY: build_binary
build_binary: $(build_TARGETS)
	$(error The $@ target has been deprecated.  Just use 'make build' or 'make' instead.)
#==============================================================================#
