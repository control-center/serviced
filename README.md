serviced
========

Serviced is a PaaS runtime. It allows users to create, manage and scale services
in a uniform way.


Dev Environment
---------------
Serviced is written in go. To install go, download go v1.1 from http://golang.org.
Untar the distribution to /usr/local/go. Ensure the following is in your environment

```bash
GOROOT=/usr/local/go
PATH="$PATH:$GOROOT/bin"
```

Setup a dev environment.

```bash
export GOPATH=~/mygo
mkdir $GOPATH/pkg -p
mkdir $GOPATH/src/github.com/zenoss -p
cd $GOPATH/src/github.com/zenoss
git clone git@github.com:zenoss/serviced.git
cd GOPATH/src/github.com/zenoss/serviced
make
```

