
def nested_subset(target, filters={}):
	"""
	Recursively determines if filters is a subset of target.
	"""
	if type(filters) is dict:
		for k,v in filters.iteritems():
			if k not in target:
				return False
			if type(v) is not type(target[k]):
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
				return False
			if type(fv) in [dict, list]:
				if not nested_subset(tv, fv):
					return False
			elif fv != tv:
				return False
	return True

def alter_dict(target, alterations):
	"""
	Recursively adds to or overwrites a dictionary.
	Not currently in use.
	"""
	if type(alterations) is dict:
		for k,v in alterations.iteritems():
			if k not in target:
				if type(v) is dict:
					target[k] = {}
				elif type(v) is list:
					target[k] = []
			if type(v) in [dict, list]:
				alter_dict(target[k], v)
			else:
				target[k] = v
	elif type(alterations) is list:
		for i in range(len(alterations)):
			if len(target) < i + 1:
				target.append(None)
			v = alterations[i]
			if type(v) is dict:
				target[i] = {}
			elif type(v) is list:
				target[i] = []
			if type(v) in [dict, list]:
				alter_dict(target[i], v)
			else:
				target[i] = v
