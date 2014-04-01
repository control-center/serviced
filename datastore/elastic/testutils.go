// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elastic

import (
	"time"
)

func InitElasticTest(index string) (ElasticDriver, error) {

	return InitElasticTestMappings(index, nil)
}

func InitElasticTestMappings(index string, mappingPaths map[string]string) (ElasticDriver, error) {
	driver := new("localhost", 9200, index)

	if !driver.isUp() {
		//TODO: start elastic
	}
	for name, path := range mappingPaths {
		driver.AddMappingFile(name, path)
	}
	err := driver.Initialize(time.Second * 10)
	if err != nil {
		return nil, err
	}
	return driver, nil
}
