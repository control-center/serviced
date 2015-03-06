import sys
import json
import unittest

import servicemigration as sm
sm.require(sm.version.API_VERSION)

from servicemigration.util import nested_subset

def getOutZope():
	"""
	Returns the Zope dictionary from the raw json
	written upon a commit.
	"""
	svcs = json.loads(open("tests/out.json", "r").read())
	for svc in svcs:
		if nested_subset(svc, {"Name": "Zope"}):
			return svc

class ServiceTest(unittest.TestCase):

	def test_no_change(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		sm.commit()
		a = json.loads(open("tests/v0.json", "r").read())
		b = json.loads(open("tests/out.json", "r").read())
		self.assertEqual(a, b)

	def test_set_description(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		svc.setDescription("an_unlikely-description")
		sm.commit()
		svc = getOutZope()
		self.assertEqual(svc["Description"], "an_unlikely-description")

	def test_get_services(self):
		sm._reloadServiceList()
		svcs = sm.getServices()
		self.assertEqual(len(svcs), 33)
		svcs = sm.getServices({
			"Tags": ["daemon"]
		})
		self.assertEqual(len(svcs), 16)

	def test_get_services_parent(self):
		sm._reloadServiceList()
		svcs = sm.getServices({}, {
			"Name": "Zenoss.core"
		})
		self.assertEqual(len(svcs), 14)

	def test_get_services_child(self):
		sm._reloadServiceList()
		svc = sm.getServices({}, {}, {
			"Name": "redis"
		})[0]
		self.assertEqual(svc.data["Name"], "Zenoss.core")

	def test_get_services_parent_child(self):
		sm._reloadServiceList()
		svc = sm.getServices({}, {
			"Name": "Zenoss.core"
		}, {
			"Name": "RegionServer"
		})[0]
		self.assertEqual(svc.data["Name"], "HBase")
		svcs = sm.getServices({
			"Name": "an_unlikely-name"
		}, {
			"Name": "Zenoss.core"
		}, {
			"Name": "RegionServer"
		})
		self.assertEqual(len(svcs), 0)

	def test_remove_endpoints(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		svc.removeEndpoints({
			"Purpose": "import"
		})
		sm.commit()
		svc = getOutZope()
		self.assertEqual(len(svc["Endpoints"]), 2)

	def test_remove_all_endpoints(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		svc.removeEndpoints()
		sm.commit()
		svc = getOutZope()
		self.assertEqual(len(svc["Endpoints"]), 0)

	def test_add_endpoint(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		svc.addEndpoint({
			"Name": "an_unlikely-name"
		})
		sm.commit()
		svc = getOutZope()
		for endpoint in svc["Endpoints"]:
			if endpoint["Name"] == "an_unlikely-name":
				return	
		raise ValueError("Didn't find new endpoint.")

	def test_get_endpoints(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		eps = svc.getEndpoints()
		self.assertEqual(len(eps), 9)
		eps = svc.getEndpoints({
			"Purpose": "export"
		})
		self.assertEqual(len(eps), 2)

	def test_remove_healthcheck(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		_ = svc.data["HealthChecks"]["answering"]
		svc.removeHealthCheck("answering")
		sm.commit()
		svc = getOutZope()
		try:
			_ = svc["HealthChecks"]["answering"]
		except KeyError:
			pass

	def test_add_healthcheck(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		svc.addHealthCheck("an_unlikely-name", {
			"Timeout": 0
		})
		sm.commit()
		svc = getOutZope()
		_ = svc["HealthChecks"]["an_unlikely-name"]

	def test_get_healthcheck(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		self.assertEqual(svc.getHealthCheck("answering")["Interval"], 10)

	def test_remove_volumes(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		svc.removeVolumes({
			"ResourcePath": "zenoss-custom-patches-pc"
		})
		sm.commit()
		svc = getOutZope()
		self.assertEqual(len(svc["Volumes"]), 7)

	def test_add_volume(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		svc.addVolume({
			"Owner": "an_unlikely-name:an_unlikely-name"
		})
		sm.commit()
		svc = getOutZope()
		for volume in svc["Volumes"]:
			if volume["Owner"] == "an_unlikely-name:an_unlikely-name":
				return	
		raise ValueError("Didn't find new volume.")

	def test_get_volume(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		vs = svc.getVolumes({
			"Owner": "zenoss:zenoss"
		})
		self.assertEqual(len(vs), 8)

	def test_remove_run(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		_ = svc.data["Runs"]["zendmd"]
		svc.removeRun("zendmd")
		sm.commit()
		svc = getOutZope()
		try:
			_ = svc["Runs"]["zendmd"]
		except KeyError:
			pass

	def test_add_run(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		svc.addRun("an_unlikely-name", "an_unlikely-operation")
		sm.commit()
		svc = getOutZope()
		self.assertEquals(svc["Runs"]["an_unlikely-name"], "an_unlikely-operation")

	def test_remove_logconfig(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		svc.removeLogConfigs({
			"Type": "zope_eventlog"
		})
		sm.commit()
		svc = getOutZope()
		self.assertEqual(len(svc["LogConfigs"]), 2)

	def test_add_logconfig(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		svc.addLogConfig({
			"Type": "redis"
		})
		sm.commit()
		svc = getOutZope()
		for config in svc["LogConfigs"]:
			if config["Type"] == "redis":
				return	
		raise ValueError("Didn't find new log config.")

	def test_get_logconfig(self):
		sm._reloadServiceList()
		svc = sm.getServices({
			"Name": "Zope"
		})[0]
		vs = svc.getLogConfigs({
			"Type": "zenossaudit"
		})
		self.assertEqual(len(vs), 1)

