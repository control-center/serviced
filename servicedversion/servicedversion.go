// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package servicedversion

var Version string
var Date string
var Gitbranch string
var Gitcommit string
var Giturl string
var Buildtag string

type ServicedVersion struct {
	Version string
	Date string
	Gitbranch string
	Gitcommit string
	Giturl string
	Buildtag string
}

func GetVersion() ServicedVersion {
	return ServicedVersion{
		Version,
		Date,
		Gitbranch,
		Gitcommit,
		Giturl,
		Buildtag,
	}
}
