#!/bin/bash

sudo GOBIN=$GOBIN GOPATH=$GOPATH PATH=$PATH `which godep` go test --tags=integration
