
import unittest

from servicemigration import version


major = int(version.API_VERSION.split('.')[0])
minor = int(version.API_VERSION.split('.')[1])
bugfx = int(version.API_VERSION.split('.')[2])

@version.versioned
def testfunc():
    pass


class VersionTest(unittest.TestCase):

    def test_not_versioned(self):
        try:
            testfunc()
        except RuntimeError:
            pass

    def test_versioned(self):
        version.require("1.0.0")
        testfunc()

    def test_same_version(self):
        version.require("%d.%d.%d" % (major, minor, bugfx))

    def test_small_major(self):
        try:
            version.require("%d.%d.%d" % (major-1, minor, bugfx))
        except ValueError:
            pass

    def test_small_minor(self):
        version.require("%d.%d.%d" % (major, minor-1, bugfx))

    def test_small_bugfix(self):
        version.require("%d.%d.%d" % (major, minor, bugfx-1))

    def test_large_major(self):
        try:
            version.require("%d.%d.%d" % (major+1, minor, bugfx))
        except ValueError:
            pass

    def test_large_minor(self):
        try:
            version.require("%d.%d.%d" % (major, minor + 1, bugfx))
        except ValueError:
            pass

    def test_large_bugfix(self):
        try:
            version.require("%d.%d.%d" % (major, minor, bugfx + 1))
        except ValueError:
            pass


