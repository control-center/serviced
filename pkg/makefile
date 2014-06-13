#############################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
##############################################################################

#
# RPM and DEB builder for Zenoss Serviced.
#

NAME          ?= serviced
FROMVERSION   ?= 0.3.70
VERSION       ?= 1.0.0
RELEASE_PHASE ?=
SUBPRODUCT    ?=
MAINTAINER    ="Zenoss CM <cm@zenoss.com>"

PKG_VERSION = $(VERSION)$(RELEASE_PHASE)
ifneq ("$(BUILD_NUMBER)", "")
PKG_VERSION = $(PKG_VERSION)-$(BUILD_NUMBER)
endif

DEB_PKG_VERSION=$(PKG_VERSION)
ifneq ("$(FROMVERSION)", "")
DEB_PKG_VERSION = $(FROMVERSION)+$(PKG_VERSION)
endif

ifneq ("$(SUBPRODUCT)", "")
FULL_NAME=$(NAME)-$(SUBPRODUCT)
else
FULL_NAME=$(NAME)
endif

define DESCRIPTION
Zenoss Serviced is a PaaS runtime. It allows users to create, manage and scale
services in a uniform way.
endef
export DESCRIPTION

INSTALL_DIR=/opt/serviced

.PHONY: all clean stage deb rpm
.SILENT: desc

all: desc

desc:
	echo "Usage: make deb or make rpm. Both options package $(FULL_NAME)-$(PKG_VERSION)."

# Clean staged files and produced packages
clean:
	rm -f $(FULL_NAME)*.deb
	rm -f $(FULL_NAME)*.rpm
	rm -rf pkgroot
	rm -rf build
	# we need to perform a clean here

clean_templates:
	find templates -type f ! -name 'README.txt' -exec rm {} + # remove everything under templates except README.txt

# Make root dir for packaging
pkgroot:
	mkdir -p pkgroot

# Build serviced binary
build:
	mkdir -p build/tmp
	cd ../ && make build_binary

# Stage serviced files
stage: build pkgroot
	# Create install location
	mkdir -p pkgroot$(INSTALL_DIR)

	# Stage binary
	mkdir -p pkgroot$(INSTALL_DIR)/bin
	cp ../serviced/serviced pkgroot$(INSTALL_DIR)/bin
	cp ../serviced/nsinit pkgroot$(INSTALL_DIR)/bin

	# Stage web files
	mkdir -p pkgroot$(INSTALL_DIR)/share/web
	mkdir -p pkgroot$(INSTALL_DIR)/share/shell

	# Stage environment default file
	mkdir -p pkgroot/etc/default
	cp serviced.default pkgroot/etc/default/serviced
	cp ../dao/elasticsearch/controlplane.json pkgroot$(INSTALL_DIR)/share/
	cp ../web/static pkgroot$(INSTALL_DIR)/share/web -R
	cp ../shell/static pkgroot$(INSTALL_DIR)/share/shell -R
	mkdir -p pkgroot$(INSTALL_DIR)/isvcs
	cp ../isvcs/resources/logstash/logstash.conf.in ../isvcs/resources/logstash/logstash.conf
	cp ../isvcs/resources pkgroot$(INSTALL_DIR)/isvcs -R
	cp ../isvcs/images pkgroot$(INSTALL_DIR)/ -R
	# stage any templates that are present
	mkdir -p pkgroot$(INSTALL_DIR)/templates
	rsync -a --exclude=README.txt templates/ pkgroot$(INSTALL_DIR)/templates/

	# add serviced to the path via symlink
	mkdir -p pkgroot/usr/bin/ && rm -f pkgroot/usr/bin/serviced && ln -s $(INSTALL_DIR)/bin/serviced  pkgroot/usr/bin/serviced
	ln -sf $(INSTALL_DIR)/bin/nsinit  pkgroot/usr/bin/nsinit

	# add serviced bash complete file to etc/bash_completion.d
	mkdir -p pkgroot/etc/bash_completion.d
	cp ../serviced/serviced-bash-completion.sh pkgroot/etc/bash_completion.d/serviced


stage_deb: stage
	# Stage upstart script
	mkdir -p pkgroot/etc/init
	cp serviced.upstart pkgroot/etc/init/serviced.conf

stage_rpm: stage
	# Stage systemd script
	mkdir -p pkgroot/usr/lib/systemd/system
	cp serviced.service pkgroot/usr/lib/systemd/system

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
		-C pkgroot \
		-m $(MAINTAINER) \
		--description "$$DESCRIPTION" \
		--deb-user root \
		--deb-group root \
		.

# Make an RPM
rpm: stage_rpm
	fpm \
		-n $(FULL_NAME) \
		-v $(PKG_VERSION) \
		-s dir \
		-t rpm \
		-a x86_64 \
		-C pkgroot \
		-m $(MAINTAINER) \
		--description "$$DESCRIPTION" \
		--rpm-user root \
		--rpm-group root \
		.
#		-d "supervisor" \
