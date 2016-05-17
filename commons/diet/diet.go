// Package diet provides an implementation of a Discrete Interval Encoding
// Tree. A Diet is a binary search tree where each node represents a range of
// integers. Ranges do not overlap, and each interval has maximal extent (i.e.,
// is not adjacent to another range).
//
// See https://web.engr.oregonstate.edu/~erwig/papers/Diet_JFP98.pdf
//
// This implementation is a modified version of the original that allows the
// insertion of an entire range at a time.
//
// Note: Deletion has NOT been implemented, due to lack of current need.
package diet

// Diet is a Discrete Interval Encoding Tree, allowing insertion of ranges of
// integers and fast intersection and membership calculation.
type Diet struct {
	root *dietNode
}

// NewDiet returns a new Diet.
func NewDiet() *Diet {
	return &Diet{}
}

// Insert adds a new range of integers to the tree.
func (d *Diet) Insert(x, y uint64) {
	x, y = sortBoundaries(x, y)
	d.root = insert(x, y, d.root)
}

// Balance balances the tree using the DSW algorithm. It is most efficient to
// do this after the tree is complete.
func (d *Diet) Balance() {
	if d.root != nil {
		d.root = balance(d.root)
	}
}

// Intersection finds the intersection of the range of integers specified with
// any of the members of the tree. It returns the number of members in common.
func (d *Diet) Intersection(x, y uint64) uint64 {
	x, y = sortBoundaries(x, y)
	return intersection(x, y, d.root)
}

// IntersectionAll finds the number of members in common between two Diets.
func (d *Diet) IntersectionAll(other *Diet) uint64 {
	return intersectionAll(d.root, other)
}

// Total finds the number of integers represented by this tree.
func (d *Diet) Total() uint64 {
	return total(d.root)
}

// Contains returns whether all of the range specified is contained within this
// diet.
func (d *Diet) Contains(x, y uint64) bool {
	x, y = sortBoundaries(x, y)
	return intersection(x, y, d.root) == y-x+1
}

type dietNode struct {
	min   uint64
	max   uint64
	left  *dietNode
	right *dietNode
}

func splitMax(min, max uint64, left, right *dietNode) (uint64, uint64, *dietNode) {
	if right == nil {
		return min, max, left
	}
	u, v, rprime := splitMax(right.min, right.max, right.left, right.right)
	newd := &dietNode{min, max, left, rprime}
	return u, v, newd
}

func splitMin(min, max uint64, left, right *dietNode) (uint64, uint64, *dietNode) {
	if left == nil {
		return min, max, right
	}
	u, v, lprime := splitMin(left.min, left.max, left.left, left.right)
	newd := &dietNode{min, max, lprime, right}
	return u, v, newd
}

func joinLeft(min, max uint64, left, right *dietNode) *dietNode {
	if left != nil {
		xprime, yprime, lprime := splitMax(left.min, left.max, left.left, left.right)
		if yprime+1 == min {
			return &dietNode{xprime, max, lprime, right}
		}
	}
	return &dietNode{min, max, left, right}
}

func joinRight(min, max uint64, left, right *dietNode) *dietNode {
	if right != nil {
		xprime, yprime, rprime := splitMin(right.min, right.max, right.left, right.right)
		if max+1 == xprime {
			return &dietNode{min, yprime, left, rprime}
		}
	}
	return &dietNode{min, max, left, right}
}

func insert(x, y uint64, d *dietNode) *dietNode {
	if d == nil {
		return &dietNode{x, y, nil, nil}
	}
	switch {
	case x >= d.min && y <= d.max: // Contained within. Do nothing.
		return d

	case y < d.min: // Does not overlap. Is less.
		if y+1 == d.min {
			return joinLeft(x, d.max, d.left, d.right)
		}
		return &dietNode{d.min, d.max, insert(x, y, d.left), d.right}

	case x > d.max: // Does not overlap. Is greater.
		if x == d.max+1 {
			return joinRight(d.min, y, d.left, d.right)
		}
		return &dietNode{d.min, d.max, d.left, insert(x, y, d.right)}

	case x < d.min && y <= d.max: // Overlaps on the left
		return joinLeft(x, d.max, d.left, d.right)

	case x >= d.min && y > d.max: // Overlaps on the right
		return joinRight(d.min, y, d.left, d.right)

	case x < d.min && y > d.max: // Overlaps on left and right
		left := joinLeft(x, d.max, d.left, d.right)
		return joinRight(left.min, y, left.left, left.right)
	}
	return d
}

func intersection(l, r uint64, d *dietNode) uint64 {
	if d == nil {
		return 0
	}
	if l > d.max {
		if d.right == nil {
			return 0
		}
		return intersection(l, r, d.right)
	}
	if r < d.min {
		if d.left == nil {
			return 0
		}
		return intersection(l, r, d.left)
	}
	if l >= d.min {
		if r <= d.max {
			return r - l + 1
		}
		isection := d.max - l + 1
		if d.right != nil {
			isection += intersection(d.max+1, r, d.right)
		}
		return isection
	}
	if r <= d.max {
		isection := r - d.min + 1
		if d.left != nil {
			isection += intersection(l, d.min-1, d.left)
		}
		return isection
	}
	if l <= d.min && r >= d.max {
		isection := d.max - d.min + 1
		if d.left != nil {
			isection += intersection(l, d.min-1, d.left)
		}
		if d.right != nil {
			isection += intersection(d.max+1, r, d.right)
		}
		return isection
	}
	return 0
}

func compress(root *dietNode, count int) *dietNode {
	var (
		child   *dietNode
		scanner *dietNode
		i       int
	)
	for i = 0; i < count; i++ {
		if scanner == nil {
			child = root
			root = child.right
		} else {
			child = scanner.right
			scanner.right = child.right
		}
		scanner = child.right
		child.right = scanner.left
		scanner.left = child
	}
	return root
}

// nearestPow2 calculates 2^(floor(log2(i)))
func nearestPow2(i int) int {
	r := 1
	for r <= i {
		r <<= 1
	}
	return r >> 1
}

func balance(root *dietNode) *dietNode {
	// Convert to a linked list
	tail := root
	rest := tail.right
	var size int
	for rest != nil {
		if rest.left == nil {
			tail = rest
			rest = rest.right
			size++
		} else {
			temp := rest.left
			rest.left = temp.right
			temp.right = rest
			rest = temp
			tail.right = temp
		}
	}
	// Now execute a series of rotations to balance
	leaves := size + 1 - nearestPow2(size+1)
	root = compress(root, leaves)
	size -= leaves
	for size > 1 {
		root = compress(root, size>>1)
		size >>= 1
	}
	// Return the new root
	return root
}

func intersectionAll(d *dietNode, other *Diet) uint64 {
	if d == nil {
		return 0
	}
	return other.Intersection(d.min, d.max) + intersectionAll(d.left, other) + intersectionAll(d.right, other)
}

func total(d *dietNode) uint64 {
	if d == nil {
		return 0
	}
	return d.max - d.min + 1 + total(d.left) + total(d.right)
}

func sortBoundaries(x, y uint64) (uint64, uint64) {
	if y > x {
		return x, y
	}
	return y, x
}
