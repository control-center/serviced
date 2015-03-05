
import os
import sys
import json

from version import versioned
from util import nested_subset

_serviceList = None

def _reloadServiceList():
    global _serviceList
    if os.environ.get("TEST_SERVICED_MIGRATION"):
        sys.argv = ["", "tests/v0.json", "tests/out.json"]
    if len(sys.argv) < 3:
        raise ValueError("A serviced migration script must be called with input and output filenames.")
    _serviceList = json.loads(open(sys.argv[1], 'r').read())

_reloadServiceList()

@versioned
def getServices(filters={}, parentFilters={}, childFilters={}):
    """
    Returns any service that matches the given filters.
    """
    f1 = []
    for svc in _serviceList:
        if nested_subset(svc, filters):
            f1.append(Service(svc))
    f2 = f1 if parentFilters == {} else []
    if parentFilters != {}:
        for svc in f1:
            parents = getServices({
                "ID": svc.data["ParentServiceID"]
            })
            if len(parents) > 1:
                raise ValueError("A service cannot have more than a single parent.")
            if len(parents) == 0:
                continue
            parent = parents[0]
            if nested_subset(parent.data, parentFilters):
                f2.append(svc)
    f3 = f2 if childFilters == {} else []
    if childFilters != {}:
        for svc in f2:
            children = getServices({
                "ParentServiceID": svc.data["ID"]
            })
            for child in children:
                if nested_subset(child.data, childFilters):
                    f3.append(svc)
    return f3

@versioned
def commit():
    """
    Commits changes made to a service. This should be called once
    after the last change you make to a service.
    """
    f = open(sys.argv[2], 'w')
    f.write(json.dumps(_serviceList, indent=4, sort_keys=True))
    f.close()


class Service:
    """
    Manages loading, altering, and commiting services.
    """

    @versioned
    def __init__(self, data):
        self.data = data


    @versioned
    def setDescription(self, desc):
        """
        Sets the description field of a service.
        """
        self.data["Description"] = desc

    @versioned
    def removeEndpoints(self, filters={}):
        """
        Removes any endpoints matching filters.
        """
        newEndpoints = []
        for endpoint in self.data["Endpoints"]:
            if not nested_subset(endpoint, filters):
                newEndpoints.append(endpoint)
        self.data["Endpoints"] = newEndpoints

    @versioned
    def addEndpoint(self, endpoint):
        """
        Adds an endpoint.
        """
        self.data["Endpoints"].append(endpoint)

    @versioned
    def getEndpoints(self, filters={}):
        """
        Returns a list of endpoints matching filters.
        """
        result = []
        for endpoint in self.data["Endpoints"]:
            if nested_subset(endpoint, filters):
                result.append(endpoint)
        return result

    @versioned
    def removeHealthCheck(self, name):
        """
        Removes healthcheck named name.
        """
        del self.data["HealthChecks"][name]

    @versioned
    def addHealthCheck(self, name, healthcheck):
        """
        Adds a healthcheck.
        """
        self.data["HealthChecks"][name] = healthcheck

    @versioned
    def getHealthCheck(self, name):
        """
        Returns the healthcheck named name.
        """
        return self.data["HealthChecks"][name]

    @versioned
    def removeVolumes(self, filters={}):
        """
        Removes any volumes matching filters.
        """
        newVolumes = []
        for volume in self.data["Volumes"]:
            if not nested_subset(volume, filters):
                newVolumes.append(volume)
        self.data["Volumes"] = newVolumes

    @versioned
    def addVolume(self, volume):
        """
        Adds a volume.
        """
        self.data["Volumes"].append(volume)

    @versioned
    def getVolumes(self, filters={}):
        """
        Returns a list of volumes matching filters.
        """
        result = []
        for volume in self.data["Volumes"]:
            if nested_subset(volume, filters):
                result.append(volume)
        return result

    def removeRun(self, name):
        """
        Removes run named name.
        """
        del self.data["Runs"][name]

    def addRun(self, name, run):
        """
        Adds an entry to self.data["Runs"].
        """
        self.data["Runs"][name] = run


