#*******************************************************************************
#* Copyright (C) Zenoss, Inc. 2013, 2014 all rights reserved.
#*
#* This content is made available according to terms specified in
#* License.zenoss under the directory where your Zenoss product is installed.
#*
#*******************************************************************************

.PHONY: buildgo buildimage

COMPONENT_NAMES    := es zk opentsdb logstash query consumer celery
HERE               := $(shell pwd)
UID                := $(shell id -u)
BUILD_DIR          := build
BUILD_REPO         := zenoss/isvcs_build
REPO               := zenoss/serviced-isvcs
TAG                := v10
REPO_DIR           := images
EXPORTED_FILE      := $(REPO_DIR)/$(REPO)/$(TAG).tar.gz
COMPONENT_ARCHIVES := $(foreach cname, $(COMPONENT_NAMES), $(BUILD_DIR)/$(cname).tar.gz)
EXPORT_CONTAINER_ID:= .isvcs_export_container_id
GZIP               := $(shell which pigz || which gzip)
DOCKERCFG           = $(HOME)/.dockercfg

ifeq "$(IN_DOCKER)" "1"
#
# Avoid building certain targets if it leads
# to the problematic docker-in-docker build
# scenario.
#
all: buildgo
else
all: buildgo isvcs_repo
endif

export: $(REPO_DIR)/$(REPO)/$(TAG).tar.gz

$(REPO_DIR)/$(REPO)/$(TAG).tar.gz: isvcs_repo
	mkdir -p $(REPO_DIR)/$(REPO)
	rm -f $(EXPORT_CONTAINER_ID)
	docker run --cidfile=$(EXPORT_CONTAINER_ID) -d $(REPO):$(TAG) echo ""
	docker export `cat $(EXPORT_CONTAINER_ID)` | $(GZIP) > $(EXPORTED_FILE)
	rm -f $(EXPORT_CONTAINER_ID)

buildgo:
	go build

build_repo:
	[ -n "$$(docker images -q $(BUILD_REPO) 2>/dev/null)" ] \
		|| docker pull $(BUILD_REPO) \
		|| docker build -t $(BUILD_REPO) build_img

$(REPO_DIR):
	mkdir -p $(@)

# Check that the isvcs image is locally available.  Otherwise download it.
#
#     NB:  The docker_list_cmd parses input of the form:
#
#     REPOSITORY                TAG    IMAGE ID       CREATED       VIRTUAL SIZE
#     quay.io/zenossinc/isvcs   v10    12d87b283130   2 weeks ago   1.276 GB
#     ..                        ..     ..             ..            ..
#
#     and returns a matching tag (column 2) if the desired image 
#     is found locally.
#
# In the event the image is hosted on a private docker repository,
# check for the presence of .dockercfg, otherwise game over since 
# authentication will fail.

isvcs_repo: docker_list_cmd = docker images $(REPO) 2>/dev/null | sed 's/ \{1,\}/\|/g' | cut -d'|' -f2 | grep ^$(TAG)$$
isvcs_repo: docker_pull_cmd = docker pull $(REPO):$(TAG)
isvcs_repo: | $(REPO_DIR) 
	@echo "$(docker_list_cmd)" ;\
	found_image_locally=$$(eval "$(docker_list_cmd)") ;\
	if [ -z "$${found_image_locally}" ]; then\
		echo "$(docker_pull_cmd)" ;\
		eval "$(docker_pull_cmd)" ;\
		rc=$$? ;\
		if [ $${rc} -ne 0 ] ;then \
			echo ;\
			echo "Error: Unable to docker pull $(REPO):$(TAG)" ;\
			echo "    $(docker_pull_cmd) [rc=$${rc}]" ;\
			echo ;\
			echo "Confirm that particular tagged image is on the remote docker repository." ;\
			echo "If this is a private repository, confirm you are suitably authenticated." ;\
			echo ;\
			exit 1 ;\
		else \
			found_image_locally=$$(eval "$(docker_list_cmd)");\
			if [ -z "$${found_image_locally}" ]; then\
				echo ;\
				echo "Error: Unable to find this image locally: $(REPO):$(TAG)." ;\
				echo "    $(docker_list_cmd)" ;\
				echo ;\
				echo "Please investigate.  You may need to build $(REPO):$(TAG)" ;\
				echo ;\
				found_image_locally=$(shell $(docker_list_cmd));\
				exit 1 ;\
			else \
				echo "$(REPO):$(TAG) found locally." ;\
			fi ;\
		fi; \
	else \
		echo "$(REPO):$(TAG) found locally." ;\
	fi

# TODO: fixme
#
# This fu for building isvcs_repo from scratch did not work for me.
# Getting: '/bin/sh: 1: git: not found' even though we apt-get it 
# in the build/Dockerfile.
#
# I'm decoupling it from the isvcs_repo rule until it can get some
# attention.
#
#if [ -z "$$(docker images $(REPO) 2>/dev/null | awk {'print $$2'} | grep $(TAG))" ]; then \
#	echo "$(MAKE) $(COMPONENT_ARCHIVES)"; \
#	echo "docker build -t $(REPO):$(TAG) $(BUILD_DIR)"; \
#fi

$(BUILD_DIR)/%.tar.gz: build_repo
	docker run --rm -v $(HERE)/$(*):/tmp/in -v $(HERE)/$(BUILD_DIR):/tmp/out -w /tmp/in $(BUILD_REPO) \
		bash -c "make TARGET=/tmp/out; chown -R $(UID):$(UID) /tmp/out/$(notdir $(@))"

clean:
	rm -rf $(BUILD_DIR)/*.tar.gz
	rm -f *.gz *.tar
	docker rmi $(REPO):$(TAG) >/dev/null 2>&1 || exit 0

mrclean: clean
	docker rmi $(BUILD_REPO) >/dev/null 2>&1 || exit 0
