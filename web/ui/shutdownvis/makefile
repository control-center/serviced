BUILD = build
SRC = src
WWW = www
NODE_MODULES=node_modules

JS = $(shell find src -name *.js)
ALL_SRC = $(shell find src)
WORKERS = $(shell find src/workers -name *.js)

LIVERELOAD = $(NODE_MODULES)/livereload/bin/livereload.js
CHOKIDAR = $(NODE_MODULES)/chokidar-cli/index.js
HTTP = $(NODE_MODULES)/http-server/bin/http-server
ROLLUP = $(NODE_MODULES)/rollup/bin/rollup

default: build www

build: $(WWW)/index.html $(WWW)/d3.js $(WWW)/available.json $(WWW)/workers

$(WWW)/index.html: $(SRC)/app.html $(JS) $(SRC)/app.css
	mkdir -p $(@D)
	$(ROLLUP) -f iife -o $(BUILD)/app.js -m inline $(SRC)/app.js 
	./inject.sh $(BUILD)/index.html $(SRC)/index.html $(SRC)/app.html $(SRC)/app.css $(BUILD)/app.js
	cp $(BUILD)/index.html $(WWW)

# d3
$(WWW)/d3.js:
	mkdir -p $(@D)
	cp $(NODE_MODULES)/d3/build/d3.js $@

$(WWW)/available.json:
	mkdir -p $(@D)
	cp data/available.json $(WWW)

$(WWW)/workers:
	cp -r $(SRC)/workers $(WWW)

# watch filesystem for changes and rebuild
# various pieces as needed
watch:
	$(MAKE) build
	$(MAKE) watch-all -j

# NOTE - you dont want this one, you just want watch
watch-all: livereload serve watch-src

watch-src:
	$(CHOKIDAR) $(ALL_SRC) -c "make"

serve:
	$(HTTP) $(WWW)

# start livereload server to update browser on changes
livereload:
	$(LIVERELOAD) $(WWW) -w 500 -d

clean:
	rm -rf build/*
	rm -rf www/*

.PHONY: watch watch-all watch-app clean
