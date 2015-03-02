
import sys
import json

from version import versioned


class Service:

	@versioned
	def __init__(self):
		self.svc = json.loads(open(sys.argv[1], 'r').read())

	@versioned
	def serialize(self):
		return json.dumps(self.svc, indent=4, sort_keys=True)

	@versioned
	def commit(self):
		f = open(sys.argv[2], 'w')
		f.write(self.serialize())
		f.close()

	@versioned
	def setDescription(self, desc):
		self.svc["Description"] = desc


