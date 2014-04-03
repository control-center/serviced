package main

import (
	"os"

	"github.com/zenoss/serviced/serviced/client/cmd"
)

func main() {
	cmd.Run(os.Args...)
}