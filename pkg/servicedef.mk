#############################################################################
#
# Copyright (C) Zenoss, Inc. 2014, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
##############################################################################

THIS_MAKEFILE := $(notdir $(CURDIR)/$(word $(words $(MAKEFILE_LIST)),$(MAKEFILE_LIST)))

#
# RPM and DEB builder for Zenoss service definitions.
#

NAME          = servicedef
FROMVERSION   = 0.3.70
VERSION       = 1.0.0
RELEASE_PHASE = 
SUBPRODUCT    = zenoss-subproduct
MAINTAINER    ="Zenoss CM <cm@zenoss.com>"
PKGROOT       = pkgroot_$(NAME)

ifneq ("$(BUILD_NUMBER)", "")
PKG_VERSION = $(VERSION)$(RELEASE_PHASE)-$(BUILD_NUMBER)
else
PKG_VERSION = $(VERSION)$(RELEASE_PHASE)
endif

ifneq ("$(FROMVERSION)", "")
DEB_PKG_VERSION = $(FROMVERSION)+$(PKG_VERSION)
else
DEB_PKG_VERSION=$(PKG_VERSION)
endif

ifneq ("$(SUBPRODUCT)", "")
FULL_NAME=$(NAME)-$(SUBPRODUCT)
else
FULL_NAME=$(NAME)
endif

define DESCRIPTION
These service definitions allow $(SUBPRODUCT) to be instantiated by the
Zenoss Control Center serviced application into a runtime environment that
leverages the scalability, performance, and deployent lifecycle associated
with Docker containers.
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
clean_dirs = $(PKGROOT)
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

stage_deb: 
	$(MAKE) -f $(THIS_MAKEFILE) clean clean_dirs=$(PKGROOT)
	cd ../ && $(MAKE) install DESTDIR=$(abspath $(PKGROOT)) PKG=deb INSTALL_TEMPLATES_ONLY=1

stage_rpm:
	$(MAKE) -f $(THIS_MAKEFILE) clean clean_dirs=$(PKGROOT)
	cd ../ && $(MAKE) install DESTDIR=$(abspath $(PKGROOT)) PKG=rpm INSTALL_TEMPLATES_ONLY=1

# Make a DEB
deb: stage_deb
	fpm \
		-n $(FULL_NAME) \
		-v $(DEB_PKG_VERSION) \
		-s dir \
		-d serviced \
		-t deb \
		-a noarch \
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
		-d serviced \
		-t rpm \
		-a noarch \
		-C pkgroot \
		-m $(MAINTAINER) \
		--description "$$DESCRIPTION" \
		--rpm-user root \
		--rpm-group root \
		.
