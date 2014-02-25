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
