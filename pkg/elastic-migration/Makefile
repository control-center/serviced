all: move
	rm -rf build dist elastic.spec

move: build
	mv dist/elastic .

build:
	docker run --rm -ti -v `pwd`:/opt/app-root/src centos/python-27-centos7 /opt/app-root/src/build.sh
