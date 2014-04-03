package cmd

/*
 #include <unistd.h>

 int GoIsatty(int fd) {
	 return isatty(fd);
 }

*/
import "C"

import (
	"fmt"
	"os"
	"syscall"
	"text/tabwriter"
)

func isatty(fd int) bool {
	switch C.GoIsatty(C.int(fd)) {
	case 0:
		return true
	default:
		return false
	}
}

func format(listitems []string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	tty := isatty(syscall.Stdin)

	i := 0
	for i < len(listitems) {
		if tty {
			output := make([]interface{}, 4)
			for j, _ := range output {
				if i < len(listitems) {
					output[j] = listitems[i]
				} else {
					output[j] = ""
				}
				i += 1
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", output...)
		} else {
			fmt.Fprintln(w, listitems[i])
			i += 1
		}
	}
	w.Flush()
}
