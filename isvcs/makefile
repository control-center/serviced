
default:
	go build
	cd es && make
	cd zk && make
	cd logstash && make
	cd opentsdb && make

clean:
	cd es && make clean
	cd zk && make clean
	cd logstash && make clean
	cd opentsdb && make clean


