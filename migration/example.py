
import servicemigration as sm
sm.require(sm.version.API_VERSION)

# Get a handle to the services titled "zproxy", and get 
# the zeroth one.
svc = sm.getServices({
	"Title": "zproxy"
})[0]

# Change the description.
svc.setDescription("an_unlikely-description")

# Get all the endpoints.
svc.getEndpoints()

# Get the export endpoint.
ep = svc.getEndpoints({
	"Purpose": "export"
})[0]

# Change the protocol to udp.
ep["Protocol"] = "udp"

# Remove the zope endpoint.
svc.removeEndpoints({
	"Name": "zope"
})

# Add a zope endpoint.
svc.addEndpoint({
	"AddressAssignment": {
	    "AssignmentType": "", 
	    "EndpointName": "", 
	    "HostID": "", 
	    "ID": "", 
	    "IPAddr": "", 
	    "PoolID": "", 
	    "Port": 0, 
	    "ServiceID": ""
	}, 
	"AddressConfig": {
	    "Port": 0, 
	    "Protocol": ""
	}, 
	"Application": "zope", 
	"ApplicationTemplate": "zope", 
	"Name": "zope", 
	"PortNumber": 9080, 
	"PortTemplate": "", 
	"Protocol": "tcp", 
	"Purpose": "import", 
	"VHosts": None, 
	"VirtualAddress": ""
})

# Commit the service.
sm.commit()

