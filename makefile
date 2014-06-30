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

PKG             = deb # deb | rpm
install_TARGETS = $(install_DIRS)
prefix          = /opt/serviced
sysconfdir      = /etc

build_TARGETS   = build_isvcs build_js $(logstash.conf) nsinit serviced

# Define GOPATH for containerized builds.
#
#    NB: Keep this in sync with build/Dockerfile: ENV GOPATH /go
#
docker_GOPATH = /go

serviced_SRC            = github.com/zenoss/serviced
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

.PHONY: build_binary 
build_binary: $(build_TARGETS)
	$(warning ":-[ Can we deprecate this poorly named target? [$@]")
	$(warning ":-[ We're building more than just one thing and we're building more than just binaries.")
	$(warning ":-] Why not just 'make all' or 'make serviced' if that is what you really want?")

# The presence of this file indicates that godep restore 
# has been run.  It will refresh when ./Godeps itself is updated.
Godeps_restored = .Godeps_restored
$(Godeps_restored): | $(GODEP)
$(Godeps_restored): $(Godeps)
	$(GODEP) restore
	touch $@

.PHONY: build_isvcs
build_isvcs: | $(Godeps_restored)
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

# As a dev convenience, we call both 'go build' and 'go install'
# so the current directory and $GOPATH/bin are updated
# with the built target.  This allows dev's to reference the target out
# of their GOPATH and type <goprog> instead of the laborious ./<goprog> :-)

docker_SRC = github.com/dotcloud/docker
nsinit_SRC = $(docker_SRC)/pkg/libcontainer/nsinit
nsinit: $(Godeps_restored)
	go build   $($@_SRC)
	go install $($@_SRC)

nsinit = $(GOBIN)/nsinit
$(nsinit): $(Godeps_restored)
	go install $($(@F)_SRC)

# https://www.gnu.org/software/make/manual/html_node/Force-Targets.html
#
# Force our go recipies to always fire since make doesn't 
# understand all of the target's *.go dependencies.  In this case let
# 'go build' determine if the target needs to be rebuilt.
FORCE:

serviced: $(Godeps_restored)
serviced: FORCE
	go build
	go install

serviced = $(GOBIN)/serviced
$(serviced): $(Godeps_restored)
$(serviced): FORCE
	go install

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

logstash.conf     = isvcs/resources/logstash/logstash.conf
logstash.conf_SRC = isvcs/resources/logstash/logstash.conf.in 
$(logstash.conf): $(logstash.conf_SRC)
	cp $? $@

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


#---------------------#
# Install targets     #
#---------------------#

install_DIRS    = $(_DESTDIR)$(prefix)
install_DIRS   += $(_DESTDIR)$(prefix)/bin
install_DIRS   += $(_DESTDIR)$(prefix)/share/web
install_DIRS   += $(_DESTDIR)$(prefix)/share/shell
install_DIRS   += $(_DESTDIR)$(prefix)/isvcs
install_DIRS   += $(_DESTDIR)$(prefix)/templates
install_DIRS   += $(_DESTDIR)$(sysconfdir)/default
install_DIRS   += $(_DESTDIR)$(sysconfdir)/bash_completion.d
#---------------------#
# Distro-specific     #
#---------------------#
_PKG = $(strip $(PKG))
ifeq "$(_PKG)" "deb"
install_DIRS   += $(_DESTDIR)$(sysconfdir)/init
endif
ifeq "$(_PKG)" "rpm"
install_DIRS   += $(_DESTDIR)/usr/lib/systemd/system
endif

# Specify the stuff to install as attributes of the various
# install directories we know about.
#
# Usage:
#
#     $(dir)_TARGETS = filename
#     $(dir)_TARGETS = src_filename:dest_filename
#
$(_DESTDIR)$(prefix)/bin_TARGETS                   = serviced
$(_DESTDIR)$(prefix)/bin_TARGETS                  += nsinit
$(_DESTDIR)$(prefix)/share/web_TARGETS             = web/static:static
$(_DESTDIR)$(prefix)/share/web_TARGETS_CP_OPT      = -R
$(_DESTDIR)$(prefix)/share/shell_TARGETS           = shell/static:.
$(_DESTDIR)$(prefix)/share/shell_TARGETS_CP_OPT    = -R
$(_DESTDIR)$(prefix)/isvcs_TARGETS                 = isvcs/resources:.
$(_DESTDIR)$(prefix)/isvcs_TARGETS_CP_OPT          = -R
$(_DESTDIR)$(prefix)_TARGETS                       = isvcs/images:.
$(_DESTDIR)$(prefix)_TARGETS_CP_OPT                = -R
$(_DESTDIR)$(sysconfdir)/default_TARGETS           = pkg/serviced.default:serviced
$(_DESTDIR)$(sysconfdir)/bash_completion.d_TARGETS = serviced-bash-completion.sh:serviced
#---------------------#
# Distro-specific     #
#---------------------#
ifeq "$(_PKG)" "deb"
$(_DESTDIR)$(sysconfdir)/init_TARGETS              = pkg/serviced.upstart:serviced.conf
endif
ifeq "$(_PKG)" "rpm"
$(_DESTDIR)/usr/lib/systemd/system_TARGETS         = pkg/serviced.service:serviced.service
endif

$(install_DIRS): dir_TARGETS = $($@_TARGETS)
$(install_DIRS): cp_OPT    = $($@_TARGETS_CP_OPT)
$(install_DIRS): FORCE
	@for install_DIR in $@ ;\
	do \
		if [ ! -d "$${install_DIR}" ];then \
			echo "mkdir -p $${install_DIR}" ;\
			mkdir -p $${install_DIR};\
			rc=$$? ;\
			if [ $${rc} -ne 0 ];then \
				exit $${rc} ;\
			fi ;\
		fi ;\
		if [ -z "$(dir_TARGETS)" ];then \
			continue ;\
		else \
			for dir_FILE in $(dir_TARGETS) ;\
			do \
				case $${dir_FILE} in \
					*:*) \
						from=`echo $${dir_FILE} | cut -d: -f1`;\
						to=`echo $${dir_FILE} | cut -d: -f2` ;\
						;;\
					*) \
						from=$${dir_FILE} ;\
						to=$${dir_FILE} ;\
						;;\
				esac ;\
				if [ -e "$${from}" ];then \
					echo "cp $(cp_OPT) $${from} $${install_DIR}/$${to}" ;\
					cp $(cp_OPT) $${from} $${install_DIR}/$${to} ;\
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
		fi ;\
	done

.PHONY: install
install: $(install_TARGETS)

#---------------------#
# Packaging targets   #
#---------------------#

.PHONY: pkgs
pkgs:
	cd pkg && $(MAKE) IN_DOCKER=$(IN_DOCKER) deb rpm

.PHONY: docker_buildandpackage dockerbuildx
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
	@for target in nsinit $(nsinit) ;\
        do \
                if [ -f "$${target}" ];then \
                        rm -f $${target} ;\
			echo "rm -f $${target}" ;\
                fi ;\
        done
	if [ -d "$(GOSRC)/$(nsinit_SRC)" ];then \
		cd $(GOSRC)/$(nsinit_SRC) && go clean ;\
	fi

.PHONY: clean_serviced
clean_serviced:
	@for target in serviced $(serviced) ;\
        do \
                if [ -f "$${target}" ];then \
                        rm -f $${target} ;\
			echo "rm -f $${target}" ;\
                fi ;\
        done
	-go clean

.PHONY: clean_pkg
clean_pkg:
	cd pkg && make clean

.PHONY: clean_godeps
clean_godeps: | $(GODEP) $(Godeps)
	$(GODEP) restore && go clean -r && go clean -i github.com/zenoss/serviced/... # this cleans all dependencies
	@if [ -f "$(Godeps_restored)" ];then \
		rm -f $(Godeps_restored) ;\
		echo "rm -f $(Godeps_restored)" ;\
	fi

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
