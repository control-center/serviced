package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

var (
	treeCharset map[string]string
	treeUTF8    map[string]string
	treeASCII   map[string]string
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

	treeCharset = treeUTF8 // default charset for tree
}

// treemap is a list of node ids mapped to its respective parent
type treemap map[string][]string

// sort organizes a treemap by the number of child nodes
func (t treemap) sort() {
	for branch := range t {
		sort.Sort(&leaf{t, branch})
	}
}

// leaf is a child node of a tree map, identified by its parent node
type leaf struct {
	tmap   treemap
	branch string
}

// Len implements sort.Sort
func (l *leaf) Len() int { return len(l.tmap[l.branch]) }

// Less implements sort.Sort
func (l *leaf) Less(i, j int) bool { return len(l.tmap[l.branch][i]) < len(l.tmap[l.branch][j]) }

// Swap implements sort.Sort
func (l *leaf) Swap(i, j int) {
	l.tmap[l.branch][i], l.tmap[l.branch][j] = l.tmap[l.branch][j], l.tmap[l.branch][i]
}

// columnmap takes a row/column mapping and transforms it to a column/row mapping
type columnmap struct {
	columns [][]string
	widths  []int
}

// newcolumnmap instantiates a new column map
func newcolumnmap(columns [][]string) *columnmap {
	return new(columnmap).init(columns)
}

// init is an initializer for a column map
func (c *columnmap) init(columns [][]string) *columnmap {
	c.columns = columns
	c.widths = make([]int, len(columns))
	for i := range c.widths {
		c.widths[i] = -1
	}
	return c
}

// width calculates the max row width for a given column
func (c *columnmap) width(index int) int {
	// if the width is not defined, calculate and store the value
	if c.widths[index] < 0 {
		var w int
		for _, row := range c.columns[index] {
			if w < len(row) {
				w = len(row)
			}
		}
		c.widths[index] = w
	}
	return c.widths[index]
}

// cell splits a cell into the "cell's rows" for printing
func (c *columnmap) cell(x, y, maxwidth int) (int, []string) {
	// figure out the max width of the cell
	if w := c.width(y); w < maxwidth {
		maxwidth = w
	}

	// getHunks is a recursive call that returns the "cell's rows"
	var getHunks func(cell string, width int) []string
	getHunks = func(cell string, width int) []string {
		if strings.TrimSpace(cell) == "" {
			return []string{}
		} else if len(cell) <= width {
			return []string{cell}
		} else if idx := strings.LastIndex(cell[:width], " "); idx > -1 {
			return append([]string{cell[:idx]}, getHunks(cell[idx+1:], width)...)
		} else {
			return append([]string{cell[:width]}, getHunks(cell[width:], width)...)
		}
	}

	return maxwidth, getHunks(c.columns[y][x], maxwidth)
}

// table is the ascii table formatter
type table struct {
	writer   io.Writer
	colwidth int
	header   []string
	rows     [][]string

	paragraph []string
	islast    bool
}

// newtable instantiates a new table formatter
func newtable(writer io.Writer, header ...interface{}) *table {
	// parse the headers into strings
	headerstr := make([]string, len(header))
	for i, h := range header {
		headerstr[i] = fmt.Sprintf("%v", h)
	}
	return &table{
		writer:   writer,
		colwidth: 30,
		header:   headerstr,
	}
}

// numcols returns the max number of columns for the table
func (tbl *table) numcols() int {
	maxcols := len(tbl.header)
	for _, row := range tbl.rows {
		if len(row) > maxcols {
			maxcols = len(row)
		}
	}
	return maxcols
}

// mapcols creates a new columnmap
func (tbl *table) mapcols() *columnmap {
	cols := tbl.numcols()
	cmap := make([][]string, cols)

	appendcol := func(row []string, index int) {
		if index < len(row) {
			cmap[index] = append(cmap[index], row[index])
		} else {
			cmap[index] = append(cmap[index], "")
		}
	}

	for i := range cmap {
		appendcol(tbl.header, i)
		for _, row := range tbl.rows {
			appendcol(row, i)
		}
	}

	return newcolumnmap(cmap)
}

// printrow prints the row to the writer
func (tbl *table) printrow(cmap *columnmap, index int) {
	isHeader := (index == 0)

	// which row?
	var row []string
	if isHeader {
		row = tbl.header
	} else {
		row = tbl.rows[index-1]
	}

	// figure out cell dimensions
	height := 1
	widths := make([]int, len(row))
	cells := make([][]string, len(row))

	for i := range row {
		width, cell := cmap.cell(index, i, tbl.colwidth)
		if len(cell) > height {
			height = len(cell)
		}
		widths[i] = width
		cells[i] = cell
	}

	// print row
	for i := 0; i < height; i++ {
		rowstr := make([]string, len(row))
		for r := range row {
			if i < len(cells[r]) {
				rowstr[r] = fmt.Sprintf("%-[2]*[1]s", cells[r][i], widths[r])
			} else {
				rowstr[r] = fmt.Sprintf("%-[2]*[1]s", "", widths[r])
			}
		}
		fmt.Fprintln(tbl.writer, strings.Join(rowstr, "   "))
	}

	// print separator
	if isHeader {
		rowstr := make([]string, len(row))
		for r := range row {
			rowstr[r] = strings.Repeat("-", widths[r])
		}
		fmt.Fprintln(tbl.writer, strings.Join(rowstr, "-+-"))
	}
}

// flush flushes the output
func (tbl *table) flush() {
	cmap := tbl.mapcols()
	// print the header
	tbl.printrow(cmap, 0)

	// print the rows
	for r := range tbl.rows {
		tbl.printrow(cmap, r+1)
	}
}

// addrow adds a new row to the table
func (tbl *table) addrow(row ...interface{}) {
	rowstr := make([]string, len(row))
	for i, r := range row {
		rowstr[i] = fmt.Sprintf("%v", r)
	}
	tbl.rows = append(tbl.rows, rowstr)
}

// add treerow adds a new treerow to the table
func (tbl *table) addtreerow(row ...interface{}) {
	var idchar string
	if tbl.islast {
		idchar = treeCharset["last"]
	} else {
		idchar = treeCharset["middle"]
	}
	row[0] = fmt.Sprintf("%s%s%v", strings.Join(tbl.paragraph, ""), idchar, row[0])
	tbl.addrow(row...)
}

// indent adds an indentation for a tree row
func (tbl *table) indent() {
	if tbl.islast {
		tbl.paragraph = append(tbl.paragraph, "  ")
	} else {
		tbl.paragraph = append(tbl.paragraph, treeCharset["bar"])
	}
}

// dedent removes an indentation for a tree row
func (tbl *table) dedent() { tbl.paragraph = tbl.paragraph[:len(tbl.paragraph)-1] }

// format tree formats the tree for printing
func (tbl *table) formattree(tmap treemap, root string, getrow func(string) []interface{}) {
	tmap.sort()

	var next func(string)
	next = func(root string) {
		for i, node := range tmap[root] {
			tbl.islast = i+1 >= len(tmap[root])
			tbl.addtreerow(getrow(node)...)
			tbl.indent()
			next(node)
			tbl.dedent()
		}
	}
	next(root)
}