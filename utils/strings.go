package utils

// StringSliceEquals compare two string slices for equality
func StringSliceEquals(lhs []string, rhs []string) bool {
	if lhs == nil && rhs == nil {
		return true
	}

	if lhs == nil && rhs != nil {
		return false
	}

	if lhs != nil && rhs == nil {
		return false
	}

	if len(lhs) != len(rhs) {
		return false
	}

	for i := range lhs {
		if lhs[i] != rhs[i] {
			return false
		}
	}

	return true
}
