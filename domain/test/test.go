package main

import (
	"encoding/json"
	"fmt"
	"github.com/zenoss/serviced/domain"
	"time"
)

func main() {
	hc := domain.StatusCheck{
		Script:   "boo!",
		Interval: time.Second * 10,
	}
	bytes, _ := json.Marshal(hc)
	fmt.Printf("%s\n", string(bytes))
}
