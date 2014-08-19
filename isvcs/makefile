# Copyright 2014, The Serviced Authors. All rights reserved.
# Use of this source code is governed by the Apache 2.0
# license that can be found in the LICENSE file.

.PHONY: buildgo buildimage

COMPONENT_NAMES    := es_serviced es_logstash zk opentsdb logstash query consumer celery
HERE               := $(shell pwd)
UID                := $(shell id -u)
BUILD_DIR          := build
BUILD_REPO         := zenoss/isvcs_build
REPO               := zenoss/serviced-isvcs
TAG                := v15
REPO_DIR           := images
EXPORTED_FILE      := $(REPO_DIR)/$(REPO)/$(TAG).tar.gz
COMPONENT_ARCHIVES := $(foreach cname, $(COMPONENT_NAMES), $(BUILD_DIR)/$(cname).tar.gz)
EXPORT_CONTAINER_ID:= .isvcs_export_container_id
GZIP               := $(shell which pigz || which gzip)
DOCKERCFG           = $(HOME)/.dockercfg
logstash.conf     = resources/logstash/logstash.conf
logstash.conf_SRC = resources/logstash/logstash.conf.in

ifeq "$(IN_DOCKER)" "1"
#
# Avoid building certain targets if it leads
# to the problematic docker-in-docker build
# scenario.
#
all: $(logstash.conf) buildgo
else
all: $(logstash.conf) buildgo isvcs_repo
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

# build the repo locally
.PHONY: repo
repo: $(COMPONENT_ARCHIVES)
	docker build -t $(REPO):$(TAG) $(BUILD_DIR);

$(logstash.conf): $(logstash.conf_SRC)
	cp $? $@

$(REPO_DIR):
	mkdir -p $(@)

# Check that the isvcs image is locally available.  Otherwise download it.
#
#     NB:  The found_image_locally_cmd parses input of the form:
#
#     REPOSITORY                TAG    IMAGE ID       CREATED       VIRTUAL SIZE
#     quay.io/zenossinc/isvcs   v10    12d87b283130   2 weeks ago   1.276 GB
#     ..                        ..     ..             ..            ..
#
#     and returns a matching tag (column 2) if the desired image 
#     is found locally.

isvcs_repo: found_image_locally_cmd = docker images $(REPO) 2>/dev/null | sed 's/ \{1,\}/\|/g' | cut -d'|' -f2 | grep -q ^$(TAG)$$
isvcs_repo: docker_pull_cmd = docker pull $(REPO):$(TAG)
isvcs_repo: | $(REPO_DIR) 
	@echo "$(found_image_locally_cmd)" ;\
	if ! eval "$(found_image_locally_cmd)"; then\
		echo "$(docker_pull_cmd)" ;\
		eval "$(docker_pull_cmd)" ;\
		if ! eval "$(found_image_locally_cmd)"; then\
			echo "Error: Unable to docker pull $(REPO):$(TAG)" ;\
			echo ;\
			echo "Confirm that particular tagged image is on the remote docker repository." ;\
			echo "If this is a private repository, confirm you are suitably authenticated." ;\
			echo ;\
			exit 1 ;\
		fi; \
	else \
		echo "$(REPO):$(TAG) found locally." ;\
	fi

$(BUILD_DIR)/%.tar.gz:
	@[ -n "$$(docker images -q $(BUILD_REPO) 2>/dev/null)" ] \
		|| docker pull $(BUILD_REPO) \
		|| docker build -t $(BUILD_REPO) build_img
	docker run --rm -v $(HERE)/$(*):/tmp/in -v $(HERE)/$(BUILD_DIR):/tmp/out -w /tmp/in $(BUILD_REPO) \
		bash -c "make TARGET=/tmp/out; chown -R $(UID):$(UID) /tmp/out/$(notdir $(@))"

clean:
	rm -rf $(BUILD_DIR)/*.tar.gz
	rm -f *.gz *.tar
	docker rmi $(REPO):$(TAG) >/dev/null 2>&1 || exit 0

mrclean: clean
	docker rmi $(BUILD_REPO) >/dev/null 2>&1 || exit 0
