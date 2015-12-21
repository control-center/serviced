#!/bin/bash
ed -s $1 <<< $'/\"VHostList\": \[\n/PortList/s/null/\[{\"PortNumber\": 1234, \"Enabled\": true}\]\n,w'