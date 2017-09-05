#!/bin/bash

#
# This script is used to build from command line the acdc
#

export GOPATH=`pwd`
export GOBIN=$GOPATH/bin
go get github.com/Jeffail/gabs
go get github.com/fjl/go-couchdb
go get github.com/philippfranke/multipart-related/related
go install
go build -a

