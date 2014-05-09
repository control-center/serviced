
package main

import (
	"github.com/zenoss/serviced/domain"
	"encoding/json"
	"fmt"
	"time"
)

func main() {
	hc := domain.HealthCheck{
		Script: "boo!",
		Interval: time.Second * 10,
	}
	bytes, _ := json.Marshal(hc)
	fmt.Printf("%s\n", string(bytes))
}
