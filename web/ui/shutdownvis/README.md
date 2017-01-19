Emergency Shutdown Threshold Visualizer and Lucky Number Generator
-------------------
This tool fetches storage usage data from tsdb and visualizes the emergency shutdown prediction algorithm over the disk available space data. The parameters to the algorithm can be adjusted, allowing the user to dial in the right paramaters for their system.

Requirements
------------------
* A recent version of Chrome (v56 works)
* [Nodejs v6](https://nodejs.org/en/download/package-manager/#debian-and-ubuntu-based-linux-distributions) or greater
* [yarn](https://yarnpkg.com/en/docs/install)

Building
------------------

Install dependencies

    yarn install

build the library

    make

serve it

    make serve

Point your browser to `http://127.0.0.1:8080/` and you're there!

Usage
-----------------
The app requires the isvcs opentsdb URL and port number (eg: `http://uiboss:4242`), as well as at least one valid tenant id. Additionally, if this is being served from a different domain than the isvcs opentsd, opentsdb will need [CORS enabled](http://opentsdb.net/docs/build/html/api_http/index.html#cors).

Enter the tsdb URL and tenant ids (as a comma separated list for multiple tenants) into the fields in the "TSDB" controls box, and click "Query" to grab the data.

Each metric fetched from tsdb is actual available space, and is represented in the graph with a solid line. For each solid line, there is a corresponding dashed line that represents the prediction algorithms output. If the dashed line dips below the threshold, then an emergency shutdown will occur.  

Use the "Shutdown Algorithm" controls to adjust the parameters of the shutdown algorithm to dial it in so that no false positives occur. Be patient after adjusting the controls as the recalculation can take a few seconds.

Finally, the mouse can be used to pan and zoom the graph.


Usage with isvcs opentsdb
-----------------
* build the shutdownvis tool as described above
* start the shutdownvis web server: `make serve`
* enable CORS for tsdb:
  * attach to the isvcs opentsdb container `docker exec -it serviced-isvcs_opentsdb /bin/bash`
  * edit `/usr/local/serviced/resources/opentsdb/opentsdb.conf` and add `tsd.http.request.cors_domains = http://<your_ip>:8080` to the end (replace `<your_ip>` with the ip of the box you will run the shutdownvis tool on)(optionally you can just set `cors_domains` to `*` if you dont give two toots about security)
  * bounce opentsdb service `supervisorctl -c /opt/zenoss/etc/supervisor.conf restart opentsdb`
* navigate to the shutdownvis page using the URL you provided to the tsdb CORS config (`http://<your_ip>:8080`) (NOTE: `127.0.0.1` won't work with CORS)
* in the "TSDB" controls box, fill in the tsdb URL (usually `http://<CC_master>:4242`)
* in the "TSDB" controls box, enter a tenant id (or comma separated ids)
* click "Query"
* data!
