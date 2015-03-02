import sys
import json
import unittest

import servicedmigration as sdm
sdm.require(sdm.version.API_VERSION)

class ServiceTest(unittest.TestCase):

	def test_no_input_file(self):
		sys.argv = ['']
		try:
			svc = sdm.Service()
		except ValueError:
			pass

	def test_nonexistent_input_file(self):
		sys.argv = ["", "i_dont_exist", "tests/out.json"]
		try:
			svc = sdm.Service()
		except IOError:
			pass

	def test_malformed_input_file(self):
		sys.argv = ["", "tests/malformed.json", "tests/out.json"]
		try:
			svc = sdm.Service()
		except ValueError:
			pass

	def test_json_file(self):
		sys.argv = ["", "tests/v0.json", "tests/out.json"]
		svc = sdm.Service()

	def test_no_change(self):
		sys.argv = ["", "tests/v0.json", "tests/out.json"]
		svc = sdm.Service()
		svc.commit()
		a = json.loads(open("tests/v0.json", "r").read())
		b = json.loads(open("tests/out.json", "r").read())
		self.assertEqual(a, b)

	def test_set_description(self):
		sys.argv = ["", "tests/v0.json", "tests/out.json"]
		svc = sdm.Service()
		svc.setDescription("an_unlikely-description")
		svc.commit()
		svc = json.loads(open("tests/out.json", "r").read())		
		self.assertEqual(svc["Description"], "an_unlikely-description")