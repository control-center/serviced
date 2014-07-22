// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/control-center/serviced/domain"
	"time"
)

func main() {
	hc := domain.HealthCheck{
		Script:   "boo!",
		Interval: time.Second * 10,
	}
	bytes, _ := json.Marshal(hc)
	fmt.Printf("%s\n", string(bytes))
}
