Emergency Shutdown Threshold Visualizer and Lucky Number Generator
-------------------
This tool fetches storage usage data from tsdb and visualizes the emergency shutdown prediction algorithm over the disk available space data. The parameters to the algorithm can be adjusted, allowing the user to dial in the right paramaters for their system.

Requirements
------------------
* A recent version of Chrome (v56 works)
* [https://nodejs.org/en/download/package-manager/#debian-and-ubuntu-based-linux-distributions](nodejs v6) or greater
* [https://yarnpkg.com/en/docs/install](yarn)

Install dependencies

    yarn install

build the library

    make

serve it

    make serve

Point your browser to `http://127.0.0.1:8080/` and you're there!

Usage
-----------------
The app requires the isvcs opentsdb URL and port number (eg: `http://uiboss:4242`), as well as at least one valid tenant id. Additionally, if this is being served from a different domain than the isvcs opentsd, opentsdb will need [http://opentsdb.net/docs/build/html/api_http/index.html#cors](CORS enabled).

Enter the tsdb URL and tenant ids (as a comma separated list for multiple tenants) into the fields in the "TSDB" controls box, and click "Query" to grab the data.

Each metric fetched from tsdb is actual available space, and is represented in the graph with a solid line. For each solid line, there is a corresponding dashed line that represents the prediction algorithms output. If the dashed line dips below the threshold, then an emergency shutdown will occur.  

Use the "Shutdown Algorithm" controls to adjust the parameters of the shutdown algorithm to dial it in so that no false positives occur. Be patient after adjusting the controls as the recalculation can take a few seconds.

Finally, the mouse can be used to pan and zoom the graph.
