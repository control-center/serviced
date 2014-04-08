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
	"strings"
	"text/tabwriter"
)

type table struct {
	writer *tabwriter.Writer
}

func newTable(minwidth, tabwidth, padding int) *table {
	w := tabwriter.NewWriter(os.Stdout, minwidth, tabwidth, padding, '\t', 0)
	return &table{w}
}

func (t *table) PrintRow(columns ...interface{}) {
	fmt.Fprintf(t.writer, strings.Repeat("%s\t", len(columns)), columns...)
	fmt.Fprintln(t.writer)
}

func (t *table) Flush() {
	t.writer.Flush()
}

func isatty(fd int) bool {
	switch C.GoIsatty(C.int(fd)) {
	case 0:
		return true
	default:
		return false
	}
}
