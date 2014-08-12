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

TARGET := $(shell pwd)

.PHONY: default tarball

default: tarball

tarball:
	cd /opt && virtualenv celery
	/opt/celery/bin/pip install -U celery[redis] pyes python-dateutil socketIO-client
	mkdir -p /opt/celery/var/log /opt/celery/var/redis /opt/celery/etc
	cp /tmp/in/redis.conf /opt/celery/etc/redis.conf
	cp /tmp/in/celery-supervisor.conf /opt/celery/etc/supervisor.conf
	ln -s /usr/local/serviced/resources/cpcelery.py /opt/celery/lib/python2.7/site-packages/cpcelery.py
	tar czf $(TARGET)/celery.tar.gz /opt
