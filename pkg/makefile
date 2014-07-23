# Copyright 2014, The Serviced Authors. All rights reserved.
# Use of this source code is governed by the Apache 2.0
# license that can be found in the LICENSE file.

#
# RPM and DEB builder for Serviced.
#

NAME          ?= serviced
FROMVERSION   ?= 0.3.70
VERSION       ?= $(shell cat ../VERSION)
RELEASE_PHASE ?=
SUBPRODUCT    ?=
MAINTAINER    ="Zenoss CM <cm@zenoss.com>"
PKGROOT       = pkgroot
INSTALL_TEMLATES = 1
# https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/#license-specification
DEB_LICENSE   = Apache-2
# https://fedoraproject.org/wiki/Licensing:Main?rd=Licensing
RPM_LICENSE   = "ASL 2.0"
VENDOR        = "Zenoss, Inc."
URL           = https://github.com/control-center/serviced
DEB_PRIORITY  = extra
CATEGORY      = admin

ifeq "$(BUILD_NUMBER)" ""
PKG_VERSION = $(VERSION)$(RELEASE_PHASE)
else
PKG_VERSION = $(VERSION)$(RELEASE_PHASE)-$(BUILD_NUMBER)
endif

ifeq "$(FROMVERSION)" ""
DEB_PKG_VERSION=$(PKG_VERSION)
else
DEB_PKG_VERSION = $(FROMVERSION)+$(PKG_VERSION)
endif

ifeq "$(SUBPRODUCT)" ""
FULL_NAME=$(NAME)
else
FULL_NAME=$(NAME)-$(SUBPRODUCT)
endif

define DESCRIPTION
Zenoss Serviced is a PaaS runtime. It allows users to create, manage and scale
services in a uniform way.
endef
export DESCRIPTION

.PHONY: all clean deb rpm
.SILENT: desc

all: desc

desc:
	echo "Usage: make deb or make rpm. Both options package $(FULL_NAME)-$(PKG_VERSION)."

.PHONY: clean_files
clean_files:
	@for pkg in $(FULL_NAME)*.deb $(FULL_NAME)*.rpm ;\
	do \
		if [ -f "$${pkg}" ];then \
			echo "rm -f $${pkg}" ;\
			if ! rm -f $${pkg} ;then \
				echo "sudo rm -f $${pkg}" ;\
				if ! sudo rm -f $${pkg} ; then \
					echo "Warning: Unable to remove $${pkg}" ;\
					exit 1 ;\
				fi ;\
			fi ;\
		fi ;\
	done

.PHONY: clean_dirs
clean_dirs = $(PKGROOT) build
clean_dirs: 
	@for dir in $(clean_dirs) ;\
	do \
		if [ -d "$${dir}" ];then \
			echo "rm -rf $${dir}" ;\
			if ! rm -rf $${dir} ;then \
				echo "sudo rm -rf $${dir}" ;\
				if ! sudo rm -rf $${dir} ; then \
					echo "Warning: Unable to remove $${dir}" ;\
					exit 1 ;\
				fi ;\
			fi ;\
		fi ;\
	done

# Clean staged files and produced packages
.PHONY: clean
clean: clean_files clean_dirs

.PHONY: clean_templates
clean_templates:
	find templates -type f ! -name 'README.txt' -exec rm {} + # remove everything under templates except README.txt

.PHONY: mrclean
mrclean: clean clean_templates

# Make root dir for packaging
$(PKGROOT):
	mkdir -p $@

# Build serviced binary
build:
	cd ../ && $(MAKE)

stage_deb: build
	make clean_dirs clean_dirs=$(PKGROOT)
	cd ../ && $(MAKE) install DESTDIR=$(abspath $(PKGROOT)) PKG=deb INSTALL_TEMPLATES=$(INSTALL_TEMPLATES)

stage_rpm: build
	make clean_dirs clean_dirs=$(PKGROOT)
	cd ../ && $(MAKE) install DESTDIR=$(abspath $(PKGROOT)) PKG=rpm INSTALL_TEMPLATES=$(INSTALL_TEMPLATES)

# Make a DEB
# net-tools provides ifconfig, needed for VIPs
deb: stage_deb
	fpm \
		-n $(FULL_NAME) \
		-v $(DEB_PKG_VERSION)~$$(lsb_release -cs) \
		-s dir \
		-d nfs-kernel-server \
		-d net-tools \
		-d nfs-common \
		-d 'lxc-docker >= 1.0.0' \
		-d 'docker-smuggle >= 2.24' \
		-t deb \
		-a x86_64 \
		-C $(PKGROOT) \
		-m $(MAINTAINER) \
		--description "$$DESCRIPTION" \
		--deb-user root \
		--deb-group root \
		--deb-priority $(DEB_PRIORITY) \
		--license $(DEB_LICENSE) \
		--vendor $(VENDOR) \
		--url $(URL) \
		--category $(CATEGORY) \
		.

# Make an RPM
rpm: stage_rpm
	fpm \
		-n $(FULL_NAME) \
		-v $(PKG_VERSION) \
		-s dir \
		-t rpm \
		-a x86_64 \
		-C $(PKGROOT) \
		-m $(MAINTAINER) \
		--description "$$DESCRIPTION" \
		--rpm-user root \
		--rpm-group root \
		--license $(RPM_LICENSE) \
		--vendor $(VENDOR) \
		--url $(URL) \
		--category $(CATEGORY) \
		.
#		-d "supervisor" \
