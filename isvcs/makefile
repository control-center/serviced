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
REPO               := zctrl/isvcs
TAG                := v5
REPO_DIR           := images
REPO_EXISTS        := .repo_tag_exists_$(TAG)
BUILD_REPO_EXISTS  := .build_repo_exists
EXPORTED_FILE      := $(REPO_DIR)/$(REPO)/$(TAG).tar.gz
COMPONENT_ARCHIVES := $(foreach cname, $(COMPONENT_NAMES), $(BUILD_DIR)/$(cname).tar.gz)
EXPORT_CONTAINER_ID:= .isvcs_export_container_id

all: buildgo $(BUILD_REPO_EXISTS) $(REPO_DIR)/$(REPO)/$(TAG).tar.gz

$(REPO_DIR)/$(REPO)/$(TAG).tar.gz: $(REPO_EXISTS)
	mkdir -p $(REPO_DIR)/$(REPO)
	rm -f $(EXPORT_CONTAINER_ID)
	docker run -cidfile=$(EXPORT_CONTAINER_ID) -d $(REPO):$(TAG) echo ""
	docker export `cat $(EXPORT_CONTAINER_ID)` | gzip > $(EXPORTED_FILE)
	rm -f $(EXPORT_CONTAINER_ID)

buildgo:
	go build

$(BUILD_REPO_EXISTS):
	docker build -t $(BUILD_REPO) build_img
	@touch $(@)

$(REPO_DIR):
	mkdir -p $(@)

$(REPO_EXISTS): $(REPO_DIR) $(COMPONENT_ARCHIVES)
	docker build -t $(REPO):$(TAG) $(BUILD_DIR)
	docker images $(REPO) | grep $(TAG) 2>&1 > /dev/null && touch $(REPO_EXISTS)

$(BUILD_DIR)/%.tar.gz:
	docker run -rm -v $(HERE)/$(*):/tmp/in -v $(HERE)/$(BUILD_DIR):/tmp/out -w /tmp/in $(BUILD_REPO) \
		bash -c "make TARGET=/tmp/out; chown -R $(UID):$(UID) /tmp/out/$(notdir $(@))"

clean:
	rm -rf $(BUILD_DIR)/*.tar.gz
	rm -f $(BUILD_REPO_EXISTS) $(REPO_EXISTS) *.gz *.tar
	docker rmi $(REPO):$(TAG) >/dev/null 2>&1 || exit 0 

mrclean: clean
	docker rmi $(BUILD_REPO) >/dev/null 2>&1 || exit 0 

