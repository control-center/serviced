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

2. Next, follow the instructions in the dev section to create the serviced 
   binary at $GOPATH/github.com/zenoss/serviced/serviced. 

3. Copy serviced binary to a location in your path.

4. One instance of serviced will be the "master".

5. Download and install elastic search.  Location is irrelvant
   http://www.elasticsearch.org/download/
```bash
wget https://download.elasticsearch.org/elasticsearch/elasticsearch/elasticsearch-0.90.5.tar.gz
tar xzf elasticsearch-0.90.5.tar.gz
cd elasticsearch-0.90.5
./bin/elasticsearch
```

6.  Install elasticsearch document models
```bash
cd $GOPATH/src/github.com/zenoss/serviced/dao/elasticsearch
curl -XPUT http://localhost:9200/controlplane -d @controlplane.json
```

7. Start the master serviced. It can also act as an agent. 
```bash
serviced -agent -master
```

8. Register the agent to the control plane. For example, to register host foo that
   is running serviced on port 4979:
```bash
serviced add-host foo:4979
```

After all the agents are registered, serviced should be working properly. It's time
to define some services.


Usage
-----
Serviced is a platform for running services. Serviced is composed of a master
serviced process and agent processes running on each host. Each host must be registered
with the master process. To register a agent process:
```bash
serviced add-host [HOSTNAME:PORT]
```

After the hosts are registered they must be placed in to a resource pool. This is done
by first creating a pool:
```bash
serviced add-pool NAME CORE_LIMIT MEMORY_LIMIT PRIORITY
```



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
export GOBIN=~/mygo/bin
mkdir $GOPATH/pkg -p
mkdir $GOPATH/src/github.com/zenoss -p
mkdir $GOPATH/src/github.com/mattbaird -p
cd $GOPATH/src/github.com/mattbaird
git clone git@github.com:zenoss/elastigo.git
cd elasticgo
go build install
cd $GOPATH/src/github.com/zenoss
git clone git@github.com:zenoss/serviced.git
cd GOPATH/src/github.com/zenoss/serviced
make
```

Purging the elastic search store.
```bash
curl -XDELETE http://localhost:9200/controlplane
```

Creating the elastic search data model.
```bash
cd dao/elasticsearch
curl -XPUT http://localhost:9200/controlplane -d @controlplane.json
```
