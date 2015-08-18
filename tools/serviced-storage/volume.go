package main

import "fmt"

func init() {
	App.Parser.AddCommand("volume", "Volume subcommands", "Driver subcommands", &Volume{})
}

type Volume struct {
	VolumeCreate `command:"create" description:"Create a volume under the given path"`
}

type VolumeCreate struct {
	Args struct {
		Name string `description:"Name of the volume to create"`
	} `positional-args:"yes" required:"yes"`
}

func (c *VolumeCreate) Execute(args []string) error {
	fmt.Println("CREATING A VOLUME")
	return nil
}
