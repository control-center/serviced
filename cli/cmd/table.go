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
	Fields                 []string
	Padding                int
	rows                   []map[string]string
	fieldSize              map[string]int
	treeIndent             []int
	rowsAddedSinceLastDent bool
}

func NewTable(fieldString string) *Table {
	fields := strings.Split(fieldString, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}

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
	t.rowsAddedSinceLastDent = true
}
func (t *Table) IndentRow() {
	if t.rowsAddedSinceLastDent || len(t.treeIndent) == 0 {
		t.treeIndent = append(t.treeIndent, 1)
	} else { //just increment the last indentation
		t.treeIndent[len(t.treeIndent)-1]++
	}
	t.rowsAddedSinceLastDent = false
}
func (t *Table) DedentRow() {
	if t.rowsAddedSinceLastDent || len(t.treeIndent) == 0 {
		t.treeIndent = append(t.treeIndent, -1)
	} else {
		t.treeIndent[len(t.treeIndent)-1]--
	}
	t.rowsAddedSinceLastDent = false
}
func (t *Table) Print() {
	colCount := len(t.Fields)
	if colCount == 0 {
		return
	}
	// compute the padding
	padding := fmt.Sprintf("%"+fmt.Sprintf("%d", t.Padding)+"s", "")
	// compute the first column width and output
	col0width, col0rows := t.getIndents(t.Fields[0])
	// display the headers
	for i, field := range t.Fields[:colCount-1] {
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
	fmt.Printf("%-s\n", t.Fields[colCount-1])

	// display the rows
	for i, row := range t.rows {
		for j, field := range t.Fields[:colCount-1] {
			if j > 0 {
				fmt.Printf("%-"+fmt.Sprintf("%d", t.fieldSize[field])+"s"+padding, row[field])
			} else {
				fmt.Printf("%-"+fmt.Sprintf("%d", col0width)+"s"+padding, col0rows[i])
			}
		}
		fmt.Printf("%-s\n", row[t.Fields[colCount-1]])
	}
}
func (t *Table) getIndents(field string) (int, []string) {
	// determines if the row is the last parent in the tree
	isLastIndex := func(index int) bool {
		if index+1 < len(t.rows) {

			if t.treeIndent[index+1] < 0 {
				return true
			} else if t.treeIndent[index+1] == 0 {
				return false
			} else {
				level := 0
				for _, ind := range t.treeIndent[index+1 : len(t.rows)] {
					if level = level + ind; level == 0 { //found another node at the same level
						return false
					} else if level < 0 { //found a node at a higher level in the tree
						return true
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
			if t.treeIndent[i+1] < 0 {
				for diff := 0; diff > t.treeIndent[i+1]; diff-- {
					size := len(indent)
					indent = strings.TrimSuffix(indent, "  ")
					if size == len(indent) {
						indent = strings.TrimSuffix(indent, treeCharset["bar"])
					}
				}
			} else if t.treeIndent[i+1] > 0 {
				if lastIndex {
					indent += "  "
				} else {
					indent += treeCharset["bar"]
				}
				for diff := 1; diff < t.treeIndent[i+1]; diff++ {
					indent += "  "
				}
			}
		}
		// compute the max width of the column
		if width := len([]rune(rows[i])); width > maxWidth { //count runes, not bytes
			maxWidth = width
		}
	}
	return maxWidth, rows
}
