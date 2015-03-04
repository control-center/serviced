import sys
import json
import unittest

import servicedmigration as sdm
sdm.require(sdm.version.API_VERSION)

class ServiceTest(unittest.TestCase):

	def test_no_change(self):
		sdm._reloadServiceList()
		svc = sdm.getServices()[0]
		sdm.commit()
		a = json.loads(open("tests/v0.json", "r").read())
		b = json.loads(open("tests/out.json", "r").read())
		self.assertEqual(a, b)

	def test_set_description(self):
		sdm._reloadServiceList()
		svc = sdm.getServices()[0]
		svc.setDescription("an_unlikely-description")
		sdm.commit()
		svc = json.loads(open("tests/out.json", "r").read())		
		self.assertEqual(svc[0]["Description"], "an_unlikely-description")

	def test_remove_endpoints(self):
		sdm._reloadServiceList()
		svc = sdm.getServices()[0]
		svc.removeEndpoints({
			"Purpose": "import"
		})
		sdm.commit()
		svc = json.loads(open("tests/out.json", "r").read())		
		self.assertEqual(len(svc[0]["Endpoints"]), 1)

	def test_add_endpoint(self):
		sdm._reloadServiceList()
		svc = sdm.getServices()[0]
		svc.addEndpoint({
			"Name": "an_unlikely-name"
		})
		sdm.commit()
		svc = json.loads(open("tests/out.json", "r").read())	
		for endpoint in svc[0]["Endpoints"]:
			if endpoint["Name"] == "an_unlikely-name":
				return	
		raise ValueError("Didn't find new endpoint.")

	def test_get_endpoints(self):
		sdm._reloadServiceList()
		svc = sdm.getServices()[0]
		eps = svc.getEndpoints()
		self.assertEqual(len(eps), 5)
		eps = svc.getEndpoints({
			"Purpose": "export"
		})
		self.assertEqual(len(eps), 1)