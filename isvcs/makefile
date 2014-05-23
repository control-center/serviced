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
BUILD_REPO         := quay.io/zenossinc/isvcs_build
REPO               := quay.io/zenossinc/isvcs
TAG                := v8
REPO_DIR           := images
EXPORTED_FILE      := $(REPO_DIR)/$(REPO)/$(TAG).tar.gz
COMPONENT_ARCHIVES := $(foreach cname, $(COMPONENT_NAMES), $(BUILD_DIR)/$(cname).tar.gz)
EXPORT_CONTAINER_ID:= .isvcs_export_container_id
GZIP               := $(shell which pigz || which gzip)

all: buildgo $(REPO_DIR)/$(REPO)/$(TAG).tar.gz

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

isvcs_repo: $(REPO_DIR) build_repo
	if [ -z "$$(docker images $(REPO) 2>/dev/null | awk {'print $$2'} | grep $(TAG))" ]; then \
		docker pull $(REPO):$(TAG); \
	fi; \
	if [ -z "$$(docker images $(REPO) 2>/dev/null | awk {'print $$2'} | grep $(TAG))" ]; then \
		$(MAKE) $(COMPONENT_ARCHIVES); \
		docker build -t $(REPO):$(TAG) $(BUILD_DIR); \
	fi

$(BUILD_DIR)/%.tar.gz:
	docker run --rm -v $(HERE)/$(*):/tmp/in -v $(HERE)/$(BUILD_DIR):/tmp/out -w /tmp/in $(BUILD_REPO) \
		bash -c "make TARGET=/tmp/out; chown -R $(UID):$(UID) /tmp/out/$(notdir $(@))"

clean:
	rm -rf $(BUILD_DIR)/*.tar.gz
	rm -f *.gz *.tar
	docker rmi $(REPO):$(TAG) >/dev/null 2>&1 || exit 0 

mrclean: clean
	docker rmi $(BUILD_REPO) >/dev/null 2>&1 || exit 0 

