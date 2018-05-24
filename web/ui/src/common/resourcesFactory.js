/* globals controlplane: true */
(function(){

    const REQUEST_TIMEOUT = 30000;
    const GET = "get";
    const PUT = "put";
    const DELETE = "delete";
    const POST = "post";

    controlplane.factory("resourcesFactory", ["$http", "$location", "$notification", "CacheFactory", "$q", "$interval", "miscUtils",
    function($http, $location, $notification, CacheFactory, $q, $interval, utils) {
        // add function to $http service to allow for noCacheGet requests
        $http.noCacheGet = function(location){
          return $http({url: location, method: "GET", params: {'time': new Date().getTime()}});
        };

        var pollingFunctions = {};

        var redirectIfUnauthorized = function(status){
          if (status === 401) {
              utils.unauthorized($location);
          }
        };

        /*
         * a methodConfig is used to create a resources
         * factory interface method. The methodConfig object
         * has the following properties:
         *
         * @prop {string} method        - GET, POST, PUT, DELETE
         * @prop {string|function} url  - a string representing the url, or a function
         *                                that can generate the url. the function will
         *                                receive arguments passed to the factory method
         * @prop {function} [payload]   - function that returns the payload to be
         *                                delivered for POST or PUT request. the function
         *                                will receive arguments passed to the factory
         *                                method
         */
        var methodConfigs = {
            assignIP: {
                method: PUT,
                url: (id, ip) => {
                  let url = `/services/${id}/ip`;
                  if (ip) {
                    url += `/${ip}`;
                  }
                  return url;
                }
            },
            getPools: {
                method: GET,
                url: "/pools",
            },
            getPool: {
                method: GET,
                url: id => `/pools/${id}`
            },
            getPoolIPs: {
                method: GET,
                url: id => `/pools/${id}/ips`
            },
            addVHost: {
                method: PUT,
                url: (serviceID, serviceName, endpointName, vhostName) => {
                    return `/services/${serviceID}/endpoint/${endpointName}/vhosts/${vhostName}`;
                },
                payload: (serviceID, serviceName) => {
                    return JSON.stringify({
                        'ServiceName': serviceName,  /* Used in messages/logs */
                    });
                }
            },
            removeVHost: {
                method: DELETE,
                url: (serviceID, endpointName, vhostName) => {
                    return `/services/${serviceID}/endpoint/${endpointName}/vhosts/${vhostName}`;
                }
            },
            enableVHost: {
                method: POST,
                url: (serviceID, serviceName, endpointName, vhostName) => {
                    return `/services/${serviceID}/endpoint/${endpointName}/vhosts/${vhostName}`;
                },
                payload: (serviceID, serviceName) => {
                    return JSON.stringify({
                        'ServiceName': serviceName,  /* Used in messages/logs */
                        'IsEnabled': true
                    });
                }
            },
            disableVHost: {
                method: POST,
                url: (serviceID, serviceName, endpointName, vhostName) => {
                    return `/services/${serviceID}/endpoint/${endpointName}/vhosts/${vhostName}`;
                },
                payload: (serviceID, serviceName) => {
                    return JSON.stringify({
                        'ServiceName': serviceName,  /* Used in messages/logs */
                        'IsEnabled': false
                    });
                }
            },
            addPort: {
                method: PUT,
                url: (serviceID, serviceName, endpointName, portName) => {
                    return `/services/${serviceID}/endpoint/${endpointName}/ports/${portName}`;
                },
                payload: (serviceID, serviceName, endpointName, portName, usetls, protocol) => {
                    return JSON.stringify({
                        'ServiceName': serviceName,  /* Used in messages/logs */
                        'UseTLS': usetls,
                        'Protocol': protocol
                    });
                }
            },
            removePort: {
                method: DELETE,
                url: (serviceID, endpointName, portName) => {
                    return `/services/${serviceID}/endpoint/${endpointName}/ports/${portName}`;
                }
            },
            enablePort: {
                method: POST,
                url: (serviceID, serviceName, endpointName, portName) => {
                    return `/services/${serviceID}/endpoint/${endpointName}/ports/${portName}`;
                },
                payload: (serviceID, serviceName) => {
                    return JSON.stringify({
                        'ServiceName': serviceName,  /* Used in messages/logs */
                        'IsEnabled': true
                    });
                }
            },
            disablePort: {
                method: POST,
                url: (serviceID, serviceName, endpointName, portName) => {
                    return `/services/${serviceID}/endpoint/${endpointName}/ports/${portName}`;
                },
                payload: (serviceID, serviceName) => {
                    return JSON.stringify({
                        'Application': serviceName,
                        'IsEnabled': false
                    });
                }
            },
            getServiceInstances: {
                method: GET,
                url: "/servicestatus"
            },
            addPool: {
                method: POST,
                url: "/pools/add",
                payload: pool => pool
            },
            removePool: {
                method: DELETE,
                url: id => `/pools/${id}`
            },
            updatePool: {
                method: PUT,
                url: id => `/pools/${id}`,
                payload: (id, pool) => pool
            },
            addPoolVirtualIP: {
                method: PUT,
                url: poolID => `/pools/${poolID}/virtualip`,
                payload: (poolID, ip, netmask, bindInterface) => {
                    return JSON.stringify({
                        PoolID: poolID,
                        IP: ip,
                        Netmask: netmask,
                        BindInterface: bindInterface
                    });
                }
            },
            removePoolVirtualIP: {
                method: DELETE,
                url: (poolID, ip) => `/pools/${poolID}/virtualip/${ip}`,
            },
            killRunning: {
                method: DELETE,
                url: (hostID, instanceID) => `/hosts/${hostID}/${instanceID}`
            },
            getHosts: {
                method: GET,
                url: "/hosts"
            },
            addHost: {
                method: POST,
                url: "/hosts/add",
                payload: host => host
            },
            getHost: {
                method: GET,
                url: id => `/hosts/${id}`
            },
            updateHost: {
                method: PUT,
                url: id => `/hosts/${id}`,
                payload: (id, host) => host
            },
            resetHostKeys: {
                method: POST,
                url: id => `/hosts/${id}/key`
            },
            removeHost: {
                method: DELETE,
                url: id => `/hosts/${id}`
            },
            getRunningHosts: {
                method: GET,
                url: "/hosts/running"
            },
            getServices: {
                method: GET,
                url: since => `/services${ since ? "?since="+ since : ""}`,
            },
            getInstanceLogs: {
                method: GET,
                url: (serviceID, instanceID) => `/services/${serviceID}/${instanceID}/logs`
            },
            dockerIsLoggedIn: {
                method: GET,
                url: "/dockerIsLoggedIn"
            },
            getAppTemplates: {
                method: GET,
                url: "/templates"
            },
            removeAppTemplate: {
                method: DELETE,
                url: id => `/templates/${id}`
            },
            updateService: {
                method: PUT,
                url: id => `/services/${id}`,
                payload: (id, service) => service
            },
            deployAppTemplate: {
                method: POST,
                url: "/templates/deploy",
                payload: template => template
            },
            removeService: {
                method: DELETE,
                url: id => `/services/${id}`
            },
            startService: {
                method: PUT,
                url: (id, skip) => `/services/${id}/startService${ skip ? "?auto=false" : "" }`
            },
            stopService: {
                method: PUT,
                url: (id, skip) => `/services/${id}/stopService${ skip ? "?auto=false" : "" }`
            },
            restartService: {
                method: PUT,
                url: (id, skip) => `/services/${id}/restartService${ skip ? "?auto=false" : "" }`
            },
            getVersion: {
                method: GET,
                url: "/version"
            },
            getDeployStatus: {
                method: POST,
                url: "/templates/deploy/status",
                payload: def => def
            },
            getDeployingTemplates: {
                method: GET,
                url: "/templates/deploy/active"
            },
            createBackup: {
                method: GET,
                url: "/backup/create"
            },
            restoreBackup: {
                method: GET,
                url: filename => `/backup/restore?filename=${filename}`
            },
            getBackupCheck: {
                method: GET,
                url: "/backup/check"
            },
            getBackupFiles: {
                method: GET,
                url: "/backup/list"
            },
            getBackupStatus: {
                method: GET,
                url: "/backup/status"
            },
            getRestoreStatus: {
                method: GET,
                url: "/backup/restore/status"
            },
            getHostAlias: {
                method: GET,
                url: "/hosts/defaultHostAlias"
            },
            getUIConfig: {
                method: GET,
                url: "/config"
            },
            getStorage: {
              method: GET,
              url: "/storage"
            }
        };

        var v2MethodConfigs = {
            getPools: {
                method: GET,
                url: () => `/api/v2/pools`,
            },
            getPoolHosts: {
                method: GET,
                url: id => `/api/v2/pools/${id}/hosts`
            },
            getHosts: {
                method: GET,
                url: () => `/api/v2/hosts`,
            },
            getHostInstances: {
                method: GET,
                url: id => `/api/v2/hosts/${id}/instances`,
            },
            getHostStatuses: {
                method: GET,
                url: ids => ids ? `/api/v2/hoststatuses?hostId=` + ids.join("&hostId=") : `/api/v2/hoststatuses`,
                ignorePending: true
            },
            getService: {
                method: GET,
                url: id => `/api/v2/services/${id}`,
            },
            getTenants: {
                method: GET,
                url: id => `/api/v2/services?tenants`,
            },
            updateService: {
                method: PUT,
                url: id => `/api/v2/services/${id}`,
                payload: (id, service) => service
            },
            getServiceAncestors: {
                method: GET,
                url: id => `/api/v2/services/${id}?ancestors`,
            },
            getServiceChildren: {
                method: GET,
                url: id => `/api/v2/services/${id}/services`,
            },
            getServiceConfig: {
                method: GET,
                url: id => `/api/v2/serviceconfigs/${id}`,
            },
            updateServiceConfig: {
                method: PUT,
                url: (id) => `/api/v2/serviceconfigs/${id}`,
                payload: (id, cfg) => {return JSON.stringify({
                    'Filename':    cfg.Filename,
                    'Owner':       cfg.Owner,
                    'Permissions': cfg.Permissions,
                    'Content':     cfg.Content
                });}
            },
            getServiceConfigs: {
                method: GET,
                url: id => `/api/v2/services/${id}/serviceconfigs`,
            },
            getServiceContext: {
                method: GET,
                url: id => `/api/v2/services/${id}/context`,
            },
            updateServiceContext: {
                method: PUT,
                url: id => `/api/v2/services/${id}/context`,
                payload: (id, ctx) => ctx
            },
            getServiceInstances: {
                method: GET,
                url: id => `/api/v2/services/${id}/instances`,
                ignorePending: true
            },
            getServiceIpAssignments: {
                method: GET,
                url: id => `/api/v2/services/${id}/ipassignments?includeChildren`,
            },
            getServiceMonitoringProfile: {
                method: GET,
                url: id => `/api/v2/services/${id}/monitoringprofile`,
            },
            getServicePublicEndpoints: {
                method: GET,
                url: id => `/api/v2/services/${id}/publicendpoints?includeChildren`,
            },
            getServiceChildPublicEndpoints: {
                method: GET,
                url: id => `/api/v2/services/${id}/publicendpoints`,
            },
            getServiceExportEndpoints: {
                method: GET,
                url: id => `/api/v2/services/${id}/exportendpoints?includeChildren`,
            },
            getServices: {
                method: GET,
                url: since => `/api/v2/services${ since ? "?since="+ since : ""}`,
            },
            getInternalServices: {
                method: GET,
                url: () => `/api/v2/internalservices`,
            },
            getInternalService: {
                method: GET,
                url: id => `/api/v2/internalservices/${id}`,
            },
            getInternalServiceInstances: {
                method: GET,
                url: id => `/api/v2/internalservices/${id}/instances`,
            },
            getInternalServiceStatuses: {
                method: GET,
                url: ids => ids ? `/api/v2/internalservicestatuses?${ids.map(id => `id=${id}`).join('&')}` : '/api/v2/internalservicestatuses',
            },
            getServiceStatus: {
                method: GET,
                url: id => `/api/v2/statuses?serviceId=${id}`,
                ignorePending: true
            },
            getServiceStatuses: {
                method: GET,
                url: ids => `/api/v2/statuses?${ids.map(id => `serviceId=${id}`).join('&')}`,
                ignorePending: true
            },
            getDescendantCounts: {
              method: GET,
              url: id => `/api/v2/services/${id}/descendantstates`,
              ignorePending: true
            }
        };

        // adds success and error functions
        // to regular promise ala $http
        function httpify(deferred){
            deferred.promise.success = function(fn){
                deferred.promise.then(fn);
                return deferred.promise;
            };
            deferred.promise.error = function(fn){
                deferred.promise.then(null, fn);
                return deferred.promise;
            };
            return deferred;
        }

        var pendingGETRequests = {};

        function generateMethod(config){
            // method should be all lowercase
            config.method = config.method.toLowerCase();

            // if url is a string, turn it into a getter fn
            if(typeof config.url === "string"){
                let url = config.url;
                config.url = () => url;
            }

            return function(/* args */){
                var url = config.url.apply(null, arguments),
                    method = config.method,
                    resourceName = url,
                    payload,
                    // deferred that will be returned to the user
                    deferred = $q.defer(),
                    requestObj;

                // if resourceName has query params, strip em off
                if(resourceName.indexOf("?")){
                    resourceName = resourceName.split("?")[0];
                }

                // NOTE: all of our code expects angular's wrapped
                // promises which provide a success and error method
                // TODO - remove the need for this
                httpify(deferred);

                // theres already a pending request to
                // this endpoint, so fail!
                if(method === GET && pendingGETRequests[resourceName]){
                    let message = `a request to ${resourceName} is pending`;
                    let data = { Detail: message };
                    deferred.reject(data);
                    return deferred.promise;
                }

                if(config.payload){
                    payload = config.payload.apply(null, arguments);
                }

                requestObj = {
                    method: method,
                    url: url,
                    headers: {
                        'Authorization': "Bearer " + window.sessionStorage.getItem("auth0AccessToken")
                    },
                    data: payload
                };

                if(method === GET){
                    requestObj.timeout = REQUEST_TIMEOUT;
                }

                $http(requestObj)
                .success(function(data, status){
                    deferred.resolve(data);
                })
                .error(function(data, status) {
                    // TODO - include status as well?
                    deferred.reject(data);
                    redirectIfUnauthorized(status);
                })
                .finally(function() {
                    if(method === GET){
                        pendingGETRequests[resourceName] = null;
                    }
                });

                // NOTE: only limits GET requests
                if(method === GET && !config.ignorePending){
                    pendingGETRequests[resourceName] = deferred;
                }

                return deferred.promise;
            };
        }

        var resourcesFactoryInterface = {
            addAppTemplate: function(fileData){
              return $http({
                  url: "/templates/add",
                  method: POST,
                  data: fileData,
                  // content-type undefined forces the browser to fill in the
                  // boundary parameter of the request
                  headers: {"Content-Type": undefined},
                  // identity returns the value it receives, which prevents
                  // transform from modifying the request in any way
                  transformRequest: angular.identity,
              }).error(function(data, status){
                  redirectIfUnauthorized(status);
              });
            },

            registerPoll: function(label, callback, interval){
              if(pollingFunctions[label] !== undefined){
                  clearInterval(pollingFunctions[label]);
              }

              pollingFunctions[label] = $interval(function(){
                  callback();
              }, interval);
            },

            unregisterAllPolls: function(){
              for(var key in pollingFunctions){
                  $interval.cancel(pollingFunctions[key]);
              }

              pollingFunctions = {};
            },

            // redirect to specific service details
            routeToService: function(id) {
                $location.path('/services/' + id);
            },

            // redirect to specific pool
            routeToPool: function(id) {
                $location.path('/pools/' + id);
            },

            // redirect to specific host
            routeToHost: function(id) {
                $location.path('/hosts/' + id);
            },

            // redirect to specific internal service details
            routeToInternalService: function(id) {
                $location.path('/internalservices/' + id);
            },

            // redirect to internal service page
            routeToInternalServices: function() {
                $location.path('/internalservices');
            },
        };

        // generate additional methods and attach
        // to interface
        for(var name in methodConfigs){
            resourcesFactoryInterface[name] = generateMethod(methodConfigs[name]);
        }

        // generate Version 2 API methods and attach
        // to interface
        resourcesFactoryInterface.v2 = {};
        for(var name in v2MethodConfigs){
            resourcesFactoryInterface.v2[name] = generateMethod(v2MethodConfigs[name]);
        }

        return resourcesFactoryInterface;
    }]);
})();
