
import sys
import json

from version import versioned


class Service:
    """
    Manages loading, altering, and commiting services.
    """

    @versioned
    def __init__(self):
        """
        Loads the json file given by sys.argv[1].
        """
        if len(sys.argv) < 3:
            raise ValueError("A serviced migration script must be called with input and output filenames.")
        self.svc = json.loads(open(sys.argv[1], 'r').read())

    @versioned
    def commit(self):
        """
        Commits changes made to a service. This should be called once
        after the last change you make to a service.
        """
        f = open(sys.argv[2], 'w')
        f.write(json.dumps(self.svc, indent=4, sort_keys=True))
        f.close()

    @versioned
    def setDescription(self, desc):
        """
        Sets the description field of a service.
        """
        self.svc["Description"] = desc


