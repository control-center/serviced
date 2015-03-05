
def nested_subset(target, filters={}):
	"""
	Recursively determines if filters is a subset of target.
	"""
	if type(filters) is dict:
		for k,v in filters.iteritems():
			if k not in target:
				return False
			if type(v) is not type(target[k]):
				if not(type(v) is str and type(target[k]) is unicode):
					return False
			if type(v) in [dict, list]:
				if not nested_subset(target[k], filters[k]):
					return False
			elif v != target[k]:
				return False
	elif type(filters) is list:
		if len(target) != len(filters):
			return False
		for i in range(len(filters)):
			fv = filters[i]
			tv = target[i]
			if type(fv) != type(tv):
				if not(type(fv) is str and type(tv) is unicode):
					return False
			if type(fv) in [dict, list]:
				if not nested_subset(tv, fv):
					return False
			elif fv != tv:
				return False
	return True
