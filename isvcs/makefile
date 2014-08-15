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

HERE               := $(shell pwd)
UID                := $(shell id -u)
REPO               := zenoss/serviced-isvcs
TAG                := v15

logstash.conf     = resources/logstash/logstash.conf
logstash.conf_SRC = resources/logstash/logstash.conf.in

ifeq "$(IN_DOCKER)" "1"
#
# Avoid building certain targets if it leads
# to the problematic docker-in-docker build
# scenario.
#
all: $(logstash.conf)
else
all: $(logstash.conf) isvcs_repo
endif

$(logstash.conf): $(logstash.conf_SRC)
	cp $? $@

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

clean:
	rm -f $(logstash.conf)
	docker rmi $(REPO):$(TAG) >/dev/null 2>&1 || exit 0

