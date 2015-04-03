// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"strings"
)

var (
	treeCharset map[string]string
	treeUTF8    map[string]string
	treeASCII   map[string]string
	treeSPACE   map[string]string
)

func init() {
	treeUTF8 = map[string]string{
		"bar":    "│ ",
		"middle": "├─",
		"last":   "└─",
	}

	treeASCII = map[string]string{
		"bar":    "| ",
		"middle": "|-",
		"last":   "+-",
	}

	treeSPACE = map[string]string{
		"bar":    "  ",
		"middle": "  ",
		"last":   "  ",
	}

	treeCharset = treeUTF8
}

type Table struct {
	Fields     []string
	Padding    int
	rows       []map[string]string
	fieldSize  map[string]int
	treeIndent []int
}

func NewTable(fields []string) *Table {
	return &Table{
		Fields:     fields,
		Padding:    1,
		rows:       make([]map[string]string, 0),
		fieldSize:  make(map[string]int),
		treeIndent: make([]int, 0),
	}
}
func (t *Table) AddRow(row map[string]interface{}) {
	tblrow := make(map[string]string)
	for name, value := range row {
		v := fmt.Sprintf("%v", value)
		tblrow[name] = v
		if maxWidth := len(v); t.fieldSize[name] < maxWidth {
			t.fieldSize[name] = maxWidth
		}
	}
	t.rows = append(t.rows, tblrow)
	if len(t.rows) > len(t.treeIndent) {
		t.treeIndent = append(t.treeIndent, 0)
	}
}
func (t *Table) IndentRow() {
	t.treeIndent = append(t.treeIndent, 1)
}
func (t *Table) DedentRow() {
	t.treeIndent = append(t.treeIndent, -1)
}
func (t *Table) Print() {
	if len(t.Fields) == 0 {
		return
	}
	// compute the padding
	padding := fmt.Sprintf("%"+fmt.Sprintf("%d", t.Padding)+"s", "")
	// compute the first column width and output
	col0width, col0rows := t.getIndents(t.Fields[0])
	// display the headers
	for i, field := range t.Fields {
		var fieldSize int
		if i > 0 {
			if width := len(field); t.fieldSize[field] < width {
				t.fieldSize[field] = width
			}
			fieldSize = t.fieldSize[field]
		} else {
			if width := len(field); col0width < width {
				col0width = width
			}
			fieldSize = col0width
		}
		fmt.Printf("%-"+fmt.Sprintf("%d", fieldSize)+"s"+padding, field)
	}
	fmt.Println()
	// display the rows
	for i, row := range t.rows {
		for j, field := range t.Fields {
			if j > 0 {
				fmt.Printf("%-"+fmt.Sprintf("%d", t.fieldSize[field])+"s"+padding, row[field])
			} else {
				fmt.Printf("%-"+fmt.Sprintf("%d", col0width)+"s"+padding, col0rows[i])
			}
		}
		fmt.Println()
	}
}
func (t *Table) getIndents(field string) (int, []string) {
	// determines if the row is the last parent in the tree
	isLastIndex := func(index int) bool {
		if index+1 < len(t.rows) {
			switch t.treeIndent[index+1] {
			case -1:
				return true
			case 0:
				return false
			case 1:
				level := t.treeIndent[index]
				for _, ind := range t.treeIndent[index+1 : len(t.rows)] {
					if level = level + ind; level == 0 {
						return false
					}
				}
			}
		}
		return true
	}
	maxWidth := t.fieldSize[field]
	rows := make([]string, len(t.rows))
	offset := 0
	indent := ""
	for i, row := range t.rows {
		level := t.treeIndent[i]
		if offset += level; offset <= 0 {
			rows[i] = row[field]
			offset = 0
			continue
		}
		lastIndex := isLastIndex(i)
		if lastIndex {
			rows[i] = indent + treeCharset["last"] + row[field]
		} else {
			rows[i] = indent + treeCharset["middle"] + row[field]
		}
		if i+1 < len(t.rows) {
			switch t.treeIndent[i+1] {
			case -1:
				size := len(indent)
				indent = strings.TrimSuffix(indent, "  ")
				if size == len(indent) {
					indent = strings.TrimSuffix(indent, treeCharset["bar"])
				}
			case 1:
				if lastIndex {
					indent += "  "
				} else {
					indent += treeCharset["bar"]
				}
			}
		}
		// compute the max width of the column
		if width := len(rows[i]); width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth, rows
}
