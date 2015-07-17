package volume

import (
	"os"
	"sort"
)

type FileInfoSlice []os.FileInfo

func (p FileInfoSlice) Len() int {
	return len(p)
}

func (p FileInfoSlice) Less(i, j int) bool {
	return p[i].ModTime().Before(p[j].ModTime())
}

func (p FileInfoSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p FileInfoSlice) Labels() []string {
	// This would probably be very slightly more efficient with a heap, but the
	// API would be more complicated
	sort.Sort(p)
	labels := make([]string, p.Len())
	for i, label := range p {
		labels[i] = label.Name()
	}
	return labels
}
