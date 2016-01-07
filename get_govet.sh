#!/bin/bash
#
# On Jan 7 2015, a change was made to govet which made it incompatible with GO 1.4.2
# The way 'go get' works, it always checks out the tip of the GO tools repo, which means
# all of our Jenkins builds were failing.
#
# The workaround is to checkout a specific older version of GO tools which is compatible
# with GO 1.4.2.
#
echo "Checking for correct govet version"

#
# if we don't ahve govet already, or we have the wrong version, then
# we need to get the right version.
#
go_getit=false
if [ ! -d ${GOSRC}/${GOTOOLS_SRC} ];then
	echo "GO tools not installed"
	go_getit=true
else
	cd ${GOSRC}/${GOTOOLS_SRC}
	GOVET_VERSION=`git rev-parse --verify HEAD`
	if [ "$GOVET_VERSION" != "c262de870b618eed648983aa994b03bc04641c72" ]; then
		echo "govet version $GOVET_VERSION is incorrect"
		go_getit=true
	fi
fi

if [ "$go_getit" = "true" ]; then
	echo "Retrieving govet ..."
	GOVET_SRC=${GOTOOLS_SRC}/cmd/vet
	go get ${GOVET_SRC}
	if [ $? -ne 0 ]; then
		cd ${GOSRC}/${GOTOOLS_SRC}
		git checkout c262de870b618eed648983aa994b03bc04641c72 >/dev/null 2>&1
		go get ${GOVET_SRC}
	fi
fi
