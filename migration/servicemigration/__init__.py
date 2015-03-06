
from version import require
from service import Service, getServices, commit

import os
if os.environ.get("TEST_SERVICED_MIGRATION"):
	from service import _reloadServiceList
	import util