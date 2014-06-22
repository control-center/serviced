################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################

javascript_TARGETS       = $(controlplane_JS) $(controlplane_MINIFIED_JS)
controlplane_JS          = $(javascript_BLDDIR)/controlplane.js
controlplane_MINIFIED_JS = $(javascript_BLDDIR)/controlplane.min.js

javascript_SRCDIR        = static/js
# This is bad make style, btw.  Source dir should not be used for building.
javascript_BLDDIR        = $(javascript_SRCDIR)

default: build

build: build-js
	./godep restore
	go build

test: build
	go test -i
	go test $(GOTEST_FLAGS)

build-js: $(javascript_TARGETS)

$(controlplane_JS)_SRC = $(javascript_SRCDIR)/main.js $(wildcard $(javascript_SRCDIR)/controllers/*.js)
$(controlplane_JS): $($(controlplane_JS)_SRC)
	cat $^ > $@ || rm -rf $@

# One day I'll know how to minify.
$(controlplane_MINIFIED_JS): $(controlplane_JS)
	$(warning WARNING: This is a scurrilous lie. We are not minifying anything.)
	cp $? $@

clean:
	go clean
	for target in $(javascript_TARGETS) ;\
	do \
		if [ -f "$${target}" ];then \
			rm $${target} ;\
		fi ;\
	done
