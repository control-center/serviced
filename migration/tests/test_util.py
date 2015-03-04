import unittest

from servicedmigration import util
# import util

target = {
	"A": "B",
	"B": 1,
	"C": {
		1: 2,
		3: 4
	},
	"D": [1,2,3],
	"E": {
		"A": {
			"B": "C",
			"C": [1,2,3]
		}
	}
}

class NestedSubsetTest(unittest.TestCase):

    def test_nested_subset(self):

		self.assertTrue(util.nested_subset(target, {
			"A": "B"
		}))

		self.assertFalse(util.nested_subset(target, {
			"A": "C"
		}))

		self.assertTrue(util.nested_subset(target, {
			"B": 1
		}))

		self.assertFalse(util.nested_subset(target, {
			"B": 2
		}))

		self.assertTrue(util.nested_subset(target, {
			"C": {
				1: 2,
				3: 4
			}
		}))

		self.assertFalse(util.nested_subset(target, {
			"C": {
				1: 2,
				3: "A"
			}
		}))

		self.assertTrue(util.nested_subset(target, {
			"D": [1,2,3],
		}))

		self.assertFalse(util.nested_subset(target, {
			"D": [1,2,4],
		}))

		self.assertFalse(util.nested_subset(target, {
			"D": [1,2,3,4],
		}))

		self.assertFalse(util.nested_subset(target, {
			"D": [1,2],
		}))

		self.assertTrue(util.nested_subset(target, {
			"E": {
				"A": {
					"B": "C",
				}
			}
		}))

		self.assertTrue(util.nested_subset(target, {
			"E": {
				"A": {
					"C": [1,2,3],
				}
			}
		}))

		self.assertFalse(util.nested_subset(target, {
			"E": {
				"A": {
					"B": "D",
				}
			}
		}))

		self.assertFalse(util.nested_subset(target, {
			"E": {
				"A": {
					"C": "D",
				}
			}
		}))

