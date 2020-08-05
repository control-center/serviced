serviced
========

Serviced is a PaaS runtime. It allows users to create, manage and scale services
in a uniform way.

Installation
------------
Serviced can run on a single host or across multiple hosts. First, serviced
depends on docker. The minimal version of docker should be **19.03.11** for running serviced **1.8.x**.
The minimum supported versions of Linux distributions are Ubuntu is 18.04 and CentOS 7.6.

1.  Follow the instruction at http://www.docker.io/ , to install
   it on every host that serviced will run on. Ensure docker is running.

2.  Install a generated package
      ```bash 
          sudo apt-key adv --keyserver keys.gnupg.net --recv-keys AA5A1AD7
    ￼     REPO=http://get.zenoss.io/apt/ubuntu
          sudo sh -c 'echo "deb [ arch=amd64 ] '${REPO}' trusty universe" > /etc/apt/sources.list.d/zenoss.list'
          sudo apt-get update
          sudo apt-get install -y serviced
      ```

    Or follow the steps below in the dev section to a source build.
   
3.  Elasticsearch 7.x uses a `mmapfs` directory by default to store its indices. 
    The default operating system limits on mmap counts is likely to be too low, 
    which may result in out of memory exceptions. 
    On Linux, the limits should be increased by running the following command as root:
    ```bash
    sysctl -w vm.max_map_count=262144
    ```
   
       To set this value permanently, update the `vm.max_map_count` setting in `/etc/sysctl.conf`. 
       To verify after rebooting, run `sysctl vm.max_map_count.`
   
4.  Start the service. On ubuntu,
    ```bash
    start serviced
    ```
       There will be some delay the first time serviced is started before it is ready
       for user requests. You can track the output of serviced at
       `/var/log/upstart/serviced.log`.

5.  Browse the UI at https://localhost

Usage
-----
Serviced is a platform for running services. Serviced is composed of a master
serviced process and agent processes running on each host. Each host must be registered
with the master process. To register an agent process:
```bash
serviced host add HOST:PORT RESOURCE_POOL
```

Dev Environment
---------------
Serviced is written in Go. To install Go, download `go v1.14.4` from http://golang.org.
Untar the distribution to `/usr/local/go`. If you use a different location for go, you
must set GOROOT. See the http://www.golang.org for more information. Ensure that
`$GOROOT/bin` is in you path.
There's an easier way to install in Ubuntu
```bash
sudo snap install --classic --channel=1.14/stable go
```

Add your development user to the "docker" group.
```bash
sudo usermod -G docker -a $USER
```
Ubuntu 18.04 is the typical development environment. There are additional dependencies
your install will need.
```bash
sudo apt-get install git mercurial libpam0g-dev make
```

With `$GOROOT` set and `$GOROOT/bin` in your `$PATH`, create a development workspace.
```bash
export GOPATH=~/mygo
export PATH="$PATH:$GOPATH/bin"
mkdir -p $GOPATH/{bin,pkg,src}
mkdir -p $GOPATH/src/github.com/control-center
cd $GOPATH/src/github.com/control-center
git clone git@github.com:control-center/serviced
    # alternatively: git clone https://github.com/control-center/serviced
cd serviced
make
```

After this, a binary should exist at `$GOPATH/bin/serviced` &
`$GOPATH/src/github.com/control-center/serviced/serviced`. You can run the server with

```bash
sudo serviced -master
```
