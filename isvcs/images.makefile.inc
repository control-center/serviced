
.PHONY: docker_image 
#.PHONY	exported_file


# The following should be filled out by includer
# REPO
# TAG

REPO_DIR=../images

REPO_EXISTS=.repo_tag_exists
EXPORTED_FILE=$(REPO_DIR)/$(REPO)/$(TAG).tar

$(REPO_DIR)/$(REPO)/$(TAG).tar: $(REPO_EXISTS)
	mkdir -p $(REPO_DIR)/$(REPO)
	docker save $(REPO) > $(EXPORTED_FILE)

docker_image: $(REPO_EXISTS)

$(REPO_EXISTS):
	docker build -t $(REPO):$(TAG) . 
	docker images | grep zenoss 2>&1 > /dev/null && touch $(REPO_EXISTS)

clean:
	rm -f $(REPO_EXISTS) *.gz *.tar
	docker rmi $(REPO):$(TAG) >/dev/null 2>&1  || exit 0 

