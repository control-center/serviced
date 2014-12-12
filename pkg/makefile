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

#
# RPM and DEB and TGZ builder for Serviced.
#

NAME          ?= serviced
VERSION       ?= $(shell cat ../VERSION)
RELEASE_PHASE ?=
GIT_COMMIT   ?= $(shell ./gitstatus.sh)
SUBPRODUCT    ?=
MAINTAINER    ="Zenoss CM <cm@zenoss.com>"
PKGROOT       = pkgroot
DUID         ?= $(shell id -u)
DGID         ?= $(shell id -g)
INSTALL_TEMLATES = 1
# https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/#license-specification
DEB_LICENSE   = Apache-2
# https://fedoraproject.org/wiki/Licensing:Main?rd=Licensing
RPM_LICENSE   = "ASL 2.0"
VENDOR        = "Zenoss, Inc."
URL           = http://controlcenter.io/
DEB_PRIORITY  = extra
CATEGORY      = admin

ifeq "$(BUILD_NUMBER)" ""
PKG_VERSION = $(VERSION)$(RELEASE_PHASE)
else
PKG_VERSION = $(VERSION)$(RELEASE_PHASE)-$(BUILD_NUMBER)
endif

ifeq "$(SUBPRODUCT)" ""
FULL_NAME=$(NAME)
else
FULL_NAME=$(NAME)-$(SUBPRODUCT)
endif

define DESCRIPTION
Zenoss Serviced PaaS runtime 
Allows services to be uniformly created, scaled, and managed.
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

stage_tgz: build
	make clean_dirs clean_dirs=$(PKGROOT)
	cd ../ && $(MAKE) install DESTDIR=$(abspath $(PKGROOT)) PKG=tgz INSTALL_TEMPLATES=$(INSTALL_TEMPLATES)

# Make a DEB
# net-tools provides ifconfig, needed for VIPs
deb: stage_deb
	fpm \
		-n $(FULL_NAME) \
		-v $(PKG_VERSION)~$$(lsb_release -cs) \
		-s dir \
		-d nfs-kernel-server \
		-d net-tools \
		-d nfs-common \
		-d 'lxc-docker >= 1.3.2' \
		-d 'lxc-docker << 1.4' \
		-d logrotate \
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
		--config-files /etc/default/serviced \
		.

# Make an RPM
rpm: stage_rpm
	fpm \
		-n $(FULL_NAME) \
		-v $(PKG_VERSION) \
		-s dir \
		-d nfs-utils \
		-d net-tools \
		-d util-linux \
		-d bash-completion \
		-d 'docker >= 1.3.2' \
		-d logrotate \
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
		--after-install rpm/postinstall \
		--before-remove rpm/preuninstall \
		--after-remove rpm/postuninstall \
		--config-files /etc/default/serviced \
		.

# Make a TGZ
tgz: stage_tgz
	tar cvfz /tmp/$(FULL_NAME)-$(GIT_COMMIT).tgz -C $(PKGROOT)/ .
	chown $(DUID):$(DGID) /tmp/$(FULL_NAME)-$(GIT_COMMIT).tgz
	mv /tmp/$(FULL_NAME)-$(GIT_COMMIT).tgz .

