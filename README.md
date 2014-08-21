serviced
========

Serviced is a PaaS runtime. It allows users to create, manage and scale services
in a uniform way.


Installation
------------
Serviced can run on a single host or across multiple hosts. First, serviced
depends on docker. 

1. Follow the instruction at http://www.docker.io/ , to install 
   it on every host that serviced will run on. Ensure docker is running.

2. Install a generated package from http://apt-dev.zendev.org . Or follow
   the steps below in the dev section to a source build.

3. Start the service. On ubuntu,
```bash
sudo start serviced
```
   There will be some delay the first time serviced is started before it is ready
   for user requests. You can track the output of serviced at 
   /var/log/upstart/serviced.log.

4. Browse the UI at https://localhost

Usage
-----
Serviced is a platform for running services. Serviced is composed of a master
serviced process and agent processes running on each host. Each host must be registered
with the master process. To register a agent process with the default resource
pool:
```bash
serviced host add HOSTNAME:4979 default
```

Dev Environment
---------------
Serviced is written in go. To install go, download go v1.3 from http://golang.org.
Untar the distribution to /usr/local/go. If you use a different location for go, you
must set GOROOT. See the http://www.golang.org for more information. Ensure that 
$GOROOT/bin is in you path.

Add your development user to the "docker" group.
```bash
sudo usermod -G docker -a $USER
```
The current user session won't be affected until after logging out then logging back in.

Ubuntu 14.04 is the typical development environment. There are additional dependencies 
your install will need.
```bash
sudo apt-get install git mercurial libpam0g-dev make
```

With $GOROOT set and $GOROOT/bin in your $PATH, create a development workspace.
```bash
export GOPATH=~/mygo
export PATH="$PATH:$GOPATH/bin"
mkdir $GOPATH/{bin,pkg,src} -p
mkdir $GOPATH/src/github.com/control-center -p
cd $GOPATH/src/github.com/control-center 
git clone git@github.com:control-center/serviced
cd serviced
make
```

After this, a binary should exist at $GOPATH/bin/serviced & 
$GOPATH/src/github.com/control-center/serviced/serviced. You can run the server with

```bash
sudo serviced -agent -master
```

