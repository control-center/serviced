# Copyright 2014, The Serviced Authors. All rights reserved.
# Use of this source code is governed by the Apache 2.0
# license that can be found in the LICENSE file.

GO = $(shell which go)

javascript_TARGETS       = $(controlplane_JS) $(controlplane_MINIFIED_JS)
controlplane_JS          = $(javascript_BLDDIR)/controlplane.js
controlplane_MINIFIED_JS = $(javascript_BLDDIR)/controlplane.min.js

javascript_SRCDIR        = static/js
# This is bad make style, btw.  Source dir should not be used for building.
javascript_BLDDIR        = $(javascript_SRCDIR)

.PHONY: default
default: build

.PHONY: build 
build: build_js
	./godep restore
	go build

.PHONY: test
test: build
	go test -i ;\
	go test $(GOTEST_FLAGS)

.PHONY: build_js build-js
build_js build-js: $(javascript_TARGETS)

$(controlplane_JS)_SRC = $(javascript_SRCDIR)/main.js $(wildcard $(javascript_SRCDIR)/controllers/*.js)
$(controlplane_JS): $($(controlplane_JS)_SRC)
	cat $^ > $@ || rm -rf $@

# One day I'll know how to minify.
$(controlplane_MINIFIED_JS): $(controlplane_JS)
	$(warning WARNING: This is a scurrilous lie. We are not minifying anything.)
	cp $? $@

.PHONY: clean
clean:
	if [ ! -z "$(GO)" ];then \
		$(GO) clean ;\
	fi
	for target in $(javascript_TARGETS) ;\
	do \
		if [ -f "$${target}" ];then \
			rm -f $${target} ;\
		fi ;\
	done
