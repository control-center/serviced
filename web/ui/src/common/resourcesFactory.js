/* globals controlplane: true */
(function(){

  controlplane.factory("resourcesFactory", ["$http", "$location", "$notification", "DSCacheFactory", "$q", "$interval", "miscUtils",
  function($http, $location, $notification, DSCacheFactory, $q, $interval, utils) {
      // add function to $http service to allow for noCacheGet requests
      $http.noCacheGet = function(location){
          return $http({url: location, method: "GET", params: {'time': new Date().getTime()}});
      };

      // caches for various endpoints
      var healthcheckCache = DSCacheFactory.createCache("healthcheckCache", {
        // only 1 healthcheck exists
        capacity: 1,
        // expire every 5 seconds
        maxAge: 5000
      });

      var runningServicesCache = DSCacheFactory.createCache("runningServicesCache", {
        // store only 10 top level services (still has many children)
        capacity: 10,
        // expire every 5 seconds
        maxAge: 5000
      });

      var templatesCache = DSCacheFactory.createCache("templatesCache", {
        capacity: 25,
        maxAge: 15000
      });

      var pollingFunctions = {};

      var redirectIfUnauthorized = function(status){
          if (status === 401) {
              utils.unauthorized($location);
          }
      };


    var methodConfigs = {
        assignIP: {
            method: "PUT",
            url: (id, ip) => `/services/${id}/ip/${ip}`,
        },
        getPools: {
            method: "GET",
            url: "/pools"
        },
        getPoolIPs: {
            method: "GET", 
            url: id => `/pools/${id}/ips`
        },
        addVHost: {
            method: "PUT",
            url: (serviceID, endpointName, vhostName) => {
                return `/services/${serviceID}/endpoint/${endpointName}/vhosts/${vhostName}`;
            },
            payload: (serviceID, endpointName, vhostName) => {
                return JSON.stringify({
                    'ServiceID': serviceID,
                    'Application': endpointName,
                    'VirtualHostName': vhostName
                });
            }
        },
        deleteVHost: {
            method: "DELETE",
            url: (serviceID, endpointName, vhostName) => {
                return `/services/${serviceID}/endpoint/${endpointName}/vhosts/${vhostName}`;
            }
        },
        getRunningServices: {
            method: "GET",
            url: "/running"
        },
        addPool: {
            method: "POST",
            url: "/pools/add",
            payload: pool => pool
        },
        removePool: {
            method: "DELETE",
            url: id => `/pools/${id}`
        },
        addPoolVirtualIP: {
            method: "PUT",
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
            method: "DELETE",
            url: (poolID, ip) => `/pools/${poolID}/virtualip/${ip}`,
        },
        killRunning: {
            method: "DELETE",
            url: (hostID, instanceID) => `/hosts/${hostID}/${instanceID}`
        },
        getHosts: {
            method: "GET",
            url: "/hosts"
        },
        addHost: {
            method: "POST",
            url: "/hosts/add",
            payload: host => host
        },
        removeHost: {
            method: "DELETE",
            url: id => `/hosts/${id}`
        },
        getRunningHosts: {
            method: "GET",
            url: "/hosts/running"
        },
        getServices: {
            method: "GET",
            url: since => `/services${ since ? "?since="+ since : ""}`,
        },
        getInstanceLogs: {
            method: "GET",
            url: (serviceID, instanceID) => `/services/${serviceID}/${instanceID}/logs`
        },
        dockerIsLoggedIn: {
            method: "GET",
            url: "/dockerIsLoggedIn"
        },
        getAppTemplates: {
            method: "GET",
            url: "/templates"
        }
    };

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
                payload;

            if(config.payload){
                payload = config.payload.apply(null, arguments);
            }

            return $http[config.method](url, payload)
                .error(function(data, status) {
                    redirectIfUnauthorized(status);
                });
        };
    }

    // generate methods from methodConfigs
    var methods = {};
    for(var name in methodConfigs){
        methods[name] = generateMethod(methodConfigs[name]);
    }

      return {
          assign_ip: methods.assignIP,
          get_pools: methods.getPools, 
          // TODO - remove this and calculate values from servicesFactory
          get_pool_ips: methods.getPoolIPs,
          add_vhost: methods.addVHost,
          delete_vhost: methods.deleteVHost,
          get_running_services: methods.getRunningServices,
          add_pool: methods.addPool,
          remove_pool: methods.removePool,
          add_pool_virtual_ip: methods.addPoolVirtualIP,
          remove_pool_virtual_ip: methods.removePoolVirtualIP,
          kill_running: methods.killRunning,
          get_hosts: methods.getHosts,
          add_host: methods.addHost,
          remove_host: methods.removeHost,
          get_running_hosts: methods.getRunningHosts,
          get_services: methods.getServices,
          get_service_state_logs: methods.getInstanceLogs,
          docker_is_logged_in: methods.dockerIsLoggedIn,
          get_app_templates: methods.getAppTemplates,


          add_app_template: function(fileData){
              return $http({
                  url: "/templates/add",
                  method: "POST",
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

          delete_app_template: function(templateID, callback){
              $http.delete('/templates/' + templateID).
                  success(function(data) {
                      $notification.create("Removed template", data.Detail).success();
                      callback(data);
                  }).
                  error(function(data){
                      $notification.create("Removing template failed", data.Detail).error();
                  });
          },

          /*
           * Create a new service definition.
           *
           * @param {object} service The service definition to create.
           * @param {function} callback Response passed to callback on success.
           */
          add_service: function(service, callback) {
              $http.post('/services/add', service).
                  success(function(data) {
                      $notification.create("", "Added new service").success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Adding service failed", data.Detail).error();
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Update an existing service
           *
           * @param {string} serviceId The ID of the service to update.
           * @param {object} editedService The modified service.
           */
          update_service: function(serviceId, editedService) {
              return $http.put('/services/' + serviceId, editedService).
                  error(function(data, status) {
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Deploy a service (application) template to a resource pool.
           *
           * @param {object} deployDef The template definition to deploy.
           * @param {function} callback Response passed to callback on success.
           */
          deploy_app_template: function(deployDef, callback, failCallback) {
              $http.post('/templates/deploy', deployDef).
                  success(function(data) {
                      $notification.create("", "Deployed application template").success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Deploying application template failed", data.Detail).error();
                      failCallback(data);
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Snapshot a running service
           *
           * @param {string} serviceId ID of the service to snapshot.
           * @param {function} callback Response passed to callback on success.
           */
          snapshot_service: function(serviceId, callback) {
              $http.noCacheGet('/services/' + serviceId + '/snapshot').
                  success(function(data) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Snapshot service failed", data.Detail).error();
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Remove a service definition.
           *
           * @param {string} serviceId The ID of the service to remove.
           * @param {function} callback Response passed to callback on success.
           */
          remove_service: function(serviceId, callback) {
              $http.delete('/services/' + serviceId).
                  success(function(data) {
                      $notification.create("Removed service", serviceId).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Removing service failed", data.Detail).error();
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Starts a service and all of its children
           *
           * @param {string} serviceId The ID of the service to start.
           * @param {function} callback Response passed to callback on success.
           * @param {bool} indicates if service's children should not be started
           */
          start_service: function(serviceId, callback, skipChildren) {
              var url = "/services/"+ serviceId + "/startService";

              // if children should NOT be started, set 'auto' param
              // to false
              if(skipChildren){
                url += "?auto=false";
              }

              $http.put(url).
                  success(function(data) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Was unable to start service", data.Detail).error();
                      redirectIfUnauthorized(status);
                  });
          },
          /*
           * Stops a service and all of its children
           *
           * @param {string} serviceId The ID of the service to stop.
           * @param {function} callback Response passed to callback on success.
           */
          stop_service: function(serviceId, callback, skipChildren) {
              var url = "/services/"+ serviceId + "/stopService";

              // if children should NOT be started, set 'auto' param
              // to false
              if(skipChildren){
                url += "?auto=false";
              }

              $http.put(url).
                  success(function(data) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Was unable to stop service", data.Detail).error();
                      redirectIfUnauthorized(status);
                  });
          },
          /*
           * Restart a service and all of its children
           *
           * @param {string} serviceId The ID of the service to stop.
           * @param {function} callback Response passed to callback on success.
           */
          restart_service: function(serviceId, callback, skipChildren) {
              var url = "/services/"+ serviceId + "/restartService";

              // if children should NOT be started, set 'auto' param
              // to false
              if(skipChildren){
                url += "?auto=false";
              }

              $http.put(url).
                  success(function(data) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Was unable to restart service", data.Detail).error();
                      redirectIfUnauthorized(status);
                  });
          },
          /**
           * Gets the Serviced version from the server
           */
          get_version: function(callback){
              $http.noCacheGet('/version').
                  success(function(data) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("", "Could not retrieve Serviced version from server.").warning();
                      redirectIfUnauthorized(status);
                  });
          },

          /**
           * Creates a backup file of serviced
           */
          create_backup: function(success, fail){
              fail = fail || function(){};

              $http.noCacheGet('/backup/create').
                  success(function(data) {
                      success(data);
                  }).
                  error(function(data, status) {
                      redirectIfUnauthorized(status);
                      fail(data, status);
                  });
          },

          /**
           * Restores a backup file of serviced
           */
          restore_backup: function(filename, success, fail){
              fail = fail || function(){};

              $http.get('/backup/restore?filename=' + filename).
                  success(function(data) {
                      success(data);
                  }).
                  error(function(data, status) {
                      redirectIfUnauthorized(status);
                      fail(data, status);
                  });
          },

          get_backup_files: function(callback){
              $http.noCacheGet('/backup/list').
                  success(function(data) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("", "Failed retrieving list of backup files.").error();
                      redirectIfUnauthorized(status);
                  });
          },

          get_backup_status: function(successCallback, failCallback){
              failCallback = failCallback || angular.noop;

              $http.noCacheGet('/backup/status').
                  success(function(data) {
                      successCallback(data);
                  }).
                  error(function(data, status) {
                      redirectIfUnauthorized(status);
                      failCallback(data, status);
                  });
          },

          get_restore_status: function(successCallback, failCallback){
              failCallback = failCallback || angular.noop;

              $http.noCacheGet('/backup/restore/status').
                  success(function(data) {
                      successCallback(data);
                  }).
                  error(function(data, status) {
                      redirectIfUnauthorized(status);
                      failCallback(data, status);
                  });
          },

          get_service_health: function(callback){

            var url = "/servicehealth";

            $http.get(url, { cache: healthcheckCache }).
              success(function(data) {
                  callback(data);
              }).
              error(function(data, status) {
                  redirectIfUnauthorized(status);
              });

            
          },

          get_deployed_templates: function(deploymentDefinition, callback){
            $http.post('/templates/deploy/status', deploymentDefinition).
              success(function(data) {
                  callback(data);
              });
          },

          get_active_templates: function(callback){
            $http.get('/templates/deploy/active', {cache: templatesCache}).
              success(function(data) {
                  callback(data);
              });
          },

          get_stats: function(callback){
            $http.get("/stats").
              success(function(data, status) {
                  callback(status);
              }).
              error(function(data, status) {
                  // TODO error screen
                  $notification.create("", 'serviced is not collecting stats.').error();
                  redirectIfUnauthorized(status);
                  callback(status);
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
          }
      };
  }]);

})();
