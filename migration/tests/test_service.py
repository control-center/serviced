import sys
import json
import unittest

import servicemigration as sm
sm.require(sm.version.API_VERSION)

class ServiceTest(unittest.TestCase):

	def test_no_change(self):
		sm._reloadServiceList()
		svc = sm.getServices()[0]
		sm.commit()
		a = json.loads(open("tests/v0.json", "r").read())
		b = json.loads(open("tests/out.json", "r").read())
		self.assertEqual(a, b)

	def test_set_description(self):
		sm._reloadServiceList()
		svc = sm.getServices()[0]
		svc.setDescription("an_unlikely-description")
		sm.commit()
		svc = json.loads(open("tests/out.json", "r").read())		
		self.assertEqual(svc[0]["Description"], "an_unlikely-description")

	def test_remove_endpoints(self):
		sm._reloadServiceList()
		svc = sm.getServices()[0]
		svc.removeEndpoints({
			"Purpose": "import"
		})
		sm.commit()
		svc = json.loads(open("tests/out.json", "r").read())		
		self.assertEqual(len(svc[0]["Endpoints"]), 1)

	def test_add_endpoint(self):
		sm._reloadServiceList()
		svc = sm.getServices()[0]
		svc.addEndpoint({
			"Name": "an_unlikely-name"
		})
		sm.commit()
		svc = json.loads(open("tests/out.json", "r").read())	
		for endpoint in svc[0]["Endpoints"]:
			if endpoint["Name"] == "an_unlikely-name":
				return	
		raise ValueError("Didn't find new endpoint.")

	def test_get_endpoints(self):
		sm._reloadServiceList()
		svc = sm.getServices()[0]
		eps = svc.getEndpoints()
		self.assertEqual(len(eps), 5)
		eps = svc.getEndpoints({
			"Purpose": "export"
		})
		self.assertEqual(len(eps), 1)