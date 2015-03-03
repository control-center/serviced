
API_VERSION = "1.0.0"

required = False

def require(version):
    """
    Ensure compatible versions between API and migration script.
    """
    global required
    major = int(API_VERSION.split('.')[0])
    minor = int(API_VERSION.split('.')[1])
    bugfx = int(API_VERSION.split('.')[2])
    major_req = int(version.split('.')[0])
    minor_req = int(version.split('.')[1])
    bugfx_req = int(version.split('.')[2])
    if major_req != major or minor_req > minor or bugfx_req > bugfx:
        raise ValueError("Serviced migrate API %s incompatible with requested version %s." % (API_VERSION, version))
    else:
        required = True

def versioned(func):
    """
    Decorator for all functions depending on API compatibility.
    """
    def func_wrapper(*args):
        if not required:
            raise RuntimeError("You must first call require(version_number) before using this function.")
        return func(*args)
    return func_wrapper



