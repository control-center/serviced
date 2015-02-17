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
      var cached_hosts_for_pool = {};
      var cached_app_templates;

      var _get_app_templates = function(callback) {
          $http.noCacheGet('/templates').
              success(function(data) {
                  cached_app_templates = data;
                  callback(data);
              }).
              error(function() {
                  // TODO error screen
                  redirectIfUnauthorized(status);
              });
      };

      var _get_hosts_for_pool = function(poolID, callback) {
          $http.noCacheGet('/pools/' + poolID + '/hosts').
              success(function(data) {
                  cached_hosts_for_pool[poolID] = data;
                  callback(data);
              }).
              error(function() {
                  // TODO error screen
                  redirectIfUnauthorized(status);
              });
      };

      var redirectIfUnauthorized = function(status){
          if (status === 401) {
              utils.unauthorized($location);
          }
      };

      return {

          /*
           * Assign an ip address to a service endpoint and it's children.  Leave IP parameter
           * null for automatic assignment.
           *
           * @param {serviceID} string the serviceID to assign an ip address
           * @param {ip} string ip address to assign to service, set as null for automatic assignment
           * @param {function} callback data is passed to a callback on success.
           */
          assign_ip: function(serviceID, ip) {
            var url = '/services/' + serviceID + "/ip";
            if (ip !== null) {
              url = url + "/" + ip;
            }
            return $http.put(url).
                error(function(data, status) {
                    redirectIfUnauthorized(status);
                });
          },

          get_pools: function() {
              return $http.get('/pools').
                  error(function(data, status) {
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Get a Pool
           * @param {string} poolID the pool id
           * @param {function} callback Pool data is passed to a callback on success.
           */
          get_pool: function(poolID, callback) {
              $http.noCacheGet('/pools/' + poolID).
                  success(function(data) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Get all possible ips for a resource pool
           *
           * @param {boolean} cacheOk Whether or not cached data is OK to use.
           * @param {function} callback Pool data is passed to a callback on success.
           */
          get_pool_ips: function(poolID, callback) {
              $http.noCacheGet('/pools/' + poolID + "/ips").
                  success(function(data) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      redirectIfUnauthorized(status);
                  });
          },

        /*
        * Get the list of services instances currently running for a given service.
        *
        * @param {string} serviceId The ID of the service to retrieve running instances for.
        * @param {function} callback Running services are passed to callback on success.
        */
        get_running_services_for_service: function(serviceId){

            var url = '/services/' + serviceId + '/running';

            return $http.get(url, { cache: runningServicesCache }).
                error(function(data, status) {
                    redirectIfUnauthorized(status);
                });
        },


          /*
           * Get a list of virtual hosts
           *
           * @param {function} callback virtual hosts are passed to callback on success.
           */
          get_vhosts: function(callback) {
              $http.noCacheGet('/vhosts').
                  success(function(data) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * add a virtual host,
           */
          add_vhost: function(serviceId, application, virtualhost) {
              var ep = '/services/' + serviceId + '/endpoint/' + application + '/vhosts/' + virtualhost;
              var object = {'ServiceID':serviceId, 'Application':application, 'VirtualHostName':virtualhost};
              var payload = JSON.stringify(object);

              return $http.put(ep, payload)
                  .error(function(data, status) {
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Remove a virtual host
           */
          delete_vhost: function(serviceId, application, virtualhost, callback) {
              var ep = '/services/' + serviceId + '/endpoint/' + application + '/vhosts/' + virtualhost;
              $http.delete(ep).
                  success(function(data) {
                      $notification.create("Removed virtual host", ep + data.Detail).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Unable to remove virtual hosts", ep + data.Detail).error();
                      redirectIfUnauthorized(status);
                  });
          },

        /*
        * Get the list of services currently running on a particular host.
        *
        * @param {string} hostId The ID of the host to retrieve running services for.
        * @param {function} callback Running services are passed to callback on success.
        */
        get_running_services_for_host: function(hostId) {
            return $http.get(`/hosts/${hostId}/running`)
                .error(function(data, status) {
                    redirectIfUnauthorized(status);
                });
        },


          /*
           * Get the list of all services currently running.
           *
           * @param {function} callback Running services are passed to callback on success.
           */
          get_running_services: function() {
              return $http.get("/running").
                  error(function(data, status) {
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Posts new resource pool information to the server.
           *
           * @param {object} pool New pool details to be added.
           * @param {function} callback Add result passed to callback on success.
           */
          add_pool: function(pool) {
              return $http.post('/pools/add', pool).
                  error(function(data, status) {
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Updates existing resource pool.
           *
           * @param {string} poolID Unique identifier for pool to be edited.
           * @param {object} editedPool New pool details for provided poolID.
           * @param {function} callback Update result passed to callback on success.
           */
          update_pool: function(poolID, editedPool, callback) {
              $http.put('/pools/' + poolID, editedPool).
                  success(function(data) {
                      $notification.create("Updated pool", poolID).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Updating pool failed", data.Detail).error();
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Deletes existing resource pool.
           *
           * @param {string} poolID Unique identifier for pool to be removed.
           * @param {function} callback Delete result passed to callback on success.
           */
          remove_pool: function(poolID, callback) {
              $http.delete('/pools/' + poolID).
                  success(function(data) {
                      $notification.create("Removed pool", poolID).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Removing pool failed", data.Detail).error();
                      redirectIfUnauthorized(status);
                  });
          },
          /*
           * Puts new resource pool virtual ip
           *
           * @param {string} pool id to add virtual ip
           * @param {string} ip virtual ip to add to pool
           * @param {function} callback Add result passed to callback on success.
           */
          add_pool_virtual_ip: function(pool, ip, netmask, _interface) {
              var payload = JSON.stringify( {'PoolID':pool, 'IP':ip, 'Netmask':netmask, 'BindInterface':_interface});

              return $http.put('/pools/' + pool + '/virtualip', payload)
                  .error(function(data, status) {
                      redirectIfUnauthorized(status);
                  });
          },
          /*
           * Delete resource pool virtual ip
           *
           * @param {string} pool id of pool which contains the virtual ip
           * @param {string} ip virtual ip to remove
           * @param {function} callback Add result passed to callback on success.
           */
          remove_pool_virtual_ip: function(pool, ip, callback) {
              $http.delete('/pools/' + pool + '/virtualip/' + ip).
                  success(function(data) {
                      $notification.create("Removed pool virtual ip", ip).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Remove pool virtual ip failed", data.Detail).error();
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Stop a running instance of a service.
           *
           * @param {string} serviceStateId Unique identifier for a service instance.
           * @param {function} callback Result passed to callback on success.
           */
          kill_running: function(hostId, serviceStateId) {
              return $http.delete('/hosts/' + hostId + '/' + serviceStateId)
                  .error(function(data, status) {
                      redirectIfUnauthorized(status);
                  });
          },
          
            get_hosts: function(){
                return $http.get("/hosts").
                    error(function(data, status) {
                      redirectIfUnauthorized(status);
                    });
            },

          /*
           * Get a host
           * @param {string} hostID the host id
           * @param {function} callback host data is passed to a callback on success.
           */
          get_host: function(hostID, callback) {
              $http.noCacheGet('/hosts/' + hostID).
                  success(function(data) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Unable to acquire host", data.Detail).error();
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Posts new host information to the server.
           *
           * @param {object} host New host details to be added.
           * @param {function} callback Add result passed to callback on success.
           */
          add_host: function(host) {
              return $http.post('/hosts/add', host)
                  .error(function(data, status){
                     redirectIfUnauthorized(status);
                  });
          },

          /*
           * Updates existing host.
           *
           * @param {string} hostId Unique identifier for host to be edited.
           * @param {object} editedHost New host details for provided hostId.
           * @param {function} callback Update result passed to callback on success.
           */
          update_host: function(hostId, editedHost, callback) {
              $http.put('/hosts/' + hostId, editedHost).
                  success(function(data) {
                      $notification.create("Updated host", hostId).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Updating host failed", data.Detail).error();
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Deletes existing host.
           *
           * @param {string} hostId Unique identifier for host to be removed.
           * @param {function} callback Delete result passed to callback on success.
           */
          remove_host: function(hostId) {
              return $http.delete('/hosts/' + hostId)
                  .error(function(data, status){
                     redirectIfUnauthorized(status);
                  });
          },

          get_running_hosts: function(){
              return $http.get('/hosts/running')
                  .error(function(data, status){
                     redirectIfUnauthorized(status);
                  });
          },

          /*
           * Get the list of hosts belonging to a specified pool.
           *
           * @param {boolean} cacheOk Whether or not cached data is OK to use.
           * @param {string} poolID Unique identifier for pool to use.
           * @param {function} callback List of hosts pass to callback on success.
           */
          get_hosts_for_pool: function(cacheOk, poolID, callback) {
              if (cacheOk && cached_hosts_for_pool[poolID]) {
                  callback(cached_hosts_for_pool[poolID]);
              } else {
                  _get_hosts_for_pool(poolID, callback);
              }
          },

        get_services: function(since){
            var url = "/services";

            if(since){
                url += "?since="+ since;
            }

            return $http.get(url).
                error(function(data, status) {
                  redirectIfUnauthorized(status);
                });
        },

          /*
           * Retrieve some (probably not the one you want) set of logs for a
           * defined service. To get more specific logs, use
           * get_service_state_logs.
           *
           * @param {string} serviceId ID of the service to retrieve logs for.
           * @param {function} callback Log data passed to callback on success.
           */
          get_service_logs: function(serviceId, callback) {
              $http.noCacheGet('/services/' + serviceId + '/logs').
                  success(function(data) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Retrieve logs for a particular host running a particular service.
           *
           * @param {string} serviceStateId ID to retrieve logs for.
           * @param {function} callback Log data passed to callback on success.
           */
          get_service_state_logs: function(serviceId, serviceStateId, callback) {
              $http.noCacheGet('/services/' + serviceId + '/' + serviceStateId + '/logs').
                  success(function(data) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      redirectIfUnauthorized(status);
                  });
          },

          /*
           * Determine if the user is logged into Docker Hub.
           * @param {function} callback boolean passed to callback on success.          
          */

          docker_is_logged_in: function(callback) {
            $http.noCacheGet('/dockerIsLoggedIn').
            success(function(data){
              callback(data.dockerLoggedIn);
            }).
            error(function() {
              $notification.create("", "Unable to retrieve Docker Hub login status.").warning();
            });
          },

          /*
           * Retrieve all defined service (a.k.a. application) templates
           *
           * @param {boolean} cacheOk Whether or not cached data is OK to use.
           * @param {function} callback Templates passed to callback on success.
           */
          get_app_templates: function(cacheOk, callback) {
              if (cacheOk && cached_app_templates) {
                  callback(cached_app_templates);
              } else {
                  _get_app_templates(callback);
              }
          },

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
