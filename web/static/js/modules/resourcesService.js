(function(){

  controlplane.
  factory("resourcesService", ["$http", "$location", "$notification", "DSCacheFactory", "$q", "$interval",
  function($http, $location, $notification, DSCacheFactory, $q, $interval) {
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
      var cached_pools;
      var cached_hosts_for_pool = {};
      var cached_hosts;
      var cached_app_templates;
      var cached_services; // top level services only
      var cached_services_map; // map of services by by Id, with children attached
      var cached_services_updated; // timestamp of last update of cached services

      var _get_services_tree = function(callback) {
          $http.noCacheGet('/services').
              success(function(data, status) {
                  if(DEBUG) console.log('Retrieved list of services');
                  cached_services = [];
                  cached_services_map = {};
                  // Map by id
                  data.map(function(svc) {
                      cached_services_map[svc.ID] = svc;
                      // Flag internal services as such.
                      svc.isvc = svc.ID.indexOf('isvc-') != -1;
                  });
                  data.map(function(svc) {
                      if (svc.ParentServiceID !== '') {
                          var parent = cached_services_map[svc.ParentServiceID];
                          if (!parent.children) {
                              parent.children = [];
                          }
                          parent.children.push(svc);
                      } else {
                          cached_services.push(svc);
                      }
                  });
                  callback(cached_services, cached_services_map);
              }).
              error(function(data, status) {
                  // TODO error screen
                  if(DEBUG) console.log('Unable to retrieve services');
                  if (status === 401) {
                      unauthorized($location);
                  }
              });
      };

      var _get_app_templates = function(callback) {
          $http.noCacheGet('/templates').
              success(function(data, status) {
                  if(DEBUG) console.log('Retrieved list of application templates');
                  cached_app_templates = data;
                  callback(data);
              }).
              error(function(data, status) {
                  // TODO error screen
                  if(DEBUG) console.log('Unable to retrieve application templates');
                  if (status === 401) {
                      unauthorized($location);
                  }
              });
      };

      // Real implementation for acquiring list of resource pools
      var _get_pools = function(callback) {
          $http.noCacheGet('/pools').
              success(function(data, status) {
                  if(DEBUG) console.log('Retrieved list of pools');
                  cached_pools = data;
                  callback(data);
              }).
              error(function(data, status) {
                  // TODO error screen
                  if(DEBUG) console.log('Unable to retrieve list of pools');
                  if (status === 401) {
                      unauthorized($location);
                  }
              });
      };

      var _get_hosts_for_pool = function(poolID, callback) {
          $http.noCacheGet('/pools/' + poolID + '/hosts').
              success(function(data, status) {
                  if(DEBUG) console.log('Retrieved hosts for pool %s', poolID);
                  cached_hosts_for_pool[poolID] = data;
                  callback(data);
              }).
              error(function(data, status) {
                  // TODO error screen
                  if(DEBUG) console.log('Unable to retrieve hosts for pool ' + poolID);
                  if (status === 401) {
                      unauthorized($location);
                  }
              });
      };

      var _get_hosts = function(callback) {
          $http.noCacheGet('/hosts').
              success(function(data, status) {
                  if(DEBUG) console.log('Retrieved host details');
                  cached_hosts = data;
                  callback(data);
              }).
              error(function(data, status) {
                  // TODO error screen
                  if(DEBUG) console.log('Unable to retrieve host details');
                  if (status === 401) {
                      unauthorized($location);
                  }
              });
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
          assign_ip: function(serviceID, ip, callback) {
            var url = '/services/' + serviceID + "/ip";
            if (ip !== null) {
              url = url + "/" + ip;
            }
            $http.put(url).
                success(function(data, status) {
                    $notification.create("Assigned IP", ip).success();
                    if (callback) {
                      callback(data);
                    }
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("Unable to assign ip", ip).error();
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
          },

          /*
           * Get the most recently retrieved map of resource pools.
           * This will also retrieve the data if it has not yet been
           * retrieved.
           *
           * @param {boolean} cacheOk Whether or not cached data is OK to use.
           * @param {function} callback Pool data is passed to a callback on success.
           */
          get_pools: function(cacheOk, callback) {
              if (cacheOk && cached_pools) {
                  if(DEBUG) console.log('Using cached pools');
                  callback(cached_pools);
              } else {
                  _get_pools(callback);
              }
          },

          /*
           * Get a Pool
           * @param {string} poolID the pool id
           * @param {function} callback Pool data is passed to a callback on success.
           */
          get_pool: function(poolID, callback) {
              $http.noCacheGet('/pools/' + poolID).
                  success(function(data, status) {
                      if(DEBUG) console.log('Retrieved %s for %s', data, poolID);
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      if(DEBUG) console.log("Unable to acquire pool", data.Detail);
                      if (status === 401) {
                          unauthorized($location);
                      }
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
                  success(function(data, status) {
                      if(DEBUG) console.log('Retrieved %s for %s', data, poolID);
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      if(DEBUG) console.log("Unable to acquire pool", data.Detail);
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          /*
           * Get the list of services instances currently running for a given service.
           *
           * @param {string} serviceId The ID of the service to retrieve running instances for.
           * @param {function} callback Running services are passed to callback on success.
           */
          get_running_services_for_service: function(serviceId, callback) {

              var url = '/services/' + serviceId + '/running';

              $http.get(url, { cache: runningServicesCache }).
                success(function(data, status) {
                    if(DEBUG) console.log('Retrieved running services for %s', serviceId);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    if(DEBUG) console.log("Unable to acquire running services", data.Detail);
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
          },


          /*
           * Get a list of virtual hosts
           *
           * @param {function} callback virtual hosts are passed to callback on success.
           */
          get_vhosts: function(callback) {
              $http.noCacheGet('/vhosts').
                  success(function(data, status) {
                      if(DEBUG) console.log('Retrieved list of virtual hosts');
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      if(DEBUG) console.log("Unable to acquire virtual hosts", data.Detail);
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          /*
           * add a virtual host,
           */
          add_vhost: function(serviceId, application, virtualhost, callback) {
              var ep = '/services/' + serviceId + '/endpoint/' + application + '/vhosts/' + virtualhost;
              var object = {'ServiceID':serviceId, 'Application':application, 'VirtualHostName':virtualhost};
              var payload = JSON.stringify( object);
              $http.put(ep, payload).
                  success(function(data, status) {
                      $notification.create("Added virtual host", ep + data.Detail).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Unable to add virtual hosts", ep + data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          /*
           * Remove a virtual host
           */
          delete_vhost: function(serviceId, application, virtualhost, callback) {
              var ep = '/services/' + serviceId + '/endpoint/' + application + '/vhosts/' + virtualhost;
              $http.delete(ep).
                  success(function(data, status) {
                      $notification.create("Removed virtual host", ep + data.Detail).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Unable to remove virtual hosts", ep + data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          /*
           * Get the list of services currently running on a particular host.
           *
           * @param {string} hostId The ID of the host to retrieve running services for.
           * @param {function} callback Running services are passed to callback on success.
           */
          get_running_services_for_host: function(hostId, callback) {
              $http.noCacheGet('/hosts/' + hostId + '/running').
                  success(function(data, status) {
                      if(DEBUG) console.log('Retrieved running services for %s', hostId);
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      if(DEBUG) console.log("Unable to acquire running services", data.Detail);
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },


          /*
           * Get the list of all services currently running.
           *
           * @param {function} callback Running services are passed to callback on success.
           */
          get_running_services: function(callback) {
              $http.noCacheGet('/running').
                  success(function(data, status) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      if(DEBUG) console.log("Unable to acquire running services", data.Detail);
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          /*
           * Posts new resource pool information to the server.
           *
           * @param {object} pool New pool details to be added.
           * @param {function} callback Add result passed to callback on success.
           */
          add_pool: function(pool, callback) {
              if(DEBUG) console.log('Adding detail: %s', JSON.stringify(pool));
              $http.post('/pools/add', pool).
                  success(function(data, status) {
                      $notification.create("", "Added new pool").success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Adding pool failed", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
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
                  success(function(data, status) {
                      $notification.create("Updated pool", poolID).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Updating pool failed", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
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
                  success(function(data, status) {
                      $notification.create("Removed pool", poolID).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Removing pool failed", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },
          /*
           * Puts new resource pool virtual ip
           *
           * @param {string} pool id to add virtual ip
           * @param {string} ip virtual ip to add to pool
           * @param {function} callback Add result passed to callback on success.
           */
          add_pool_virtual_ip: function(pool, ip, netmask, _interface, callback) {
              var payload = JSON.stringify( {'PoolID':pool, 'IP':ip, 'Netmask':netmask, 'BindInterface':_interface});
              if(DEBUG) console.log('Adding pool virtual ip: %s', payload);
              $http.put('/pools/' + pool + '/virtualip', payload).
                  success(function(data, status) {
                      $notification.create("Added new pool virtual ip", ip).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Adding pool virtual ip failed", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
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
              if(DEBUG) console.log('Removing pool virtual ip: poolID:%s ip:%s', pool, ip);
              $http.delete('/pools/' + pool + '/virtualip/' + ip).
                  success(function(data, status) {
                      $notification.create("Removed pool virtual ip", ip).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Remove pool virtual ip failed", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          /*
           * Stop a running instance of a service.
           *
           * @param {string} serviceStateId Unique identifier for a service instance.
           * @param {function} callback Result passed to callback on success.
           */
          kill_running: function(hostId, serviceStateId, callback) {
              $http.delete('/hosts/' + hostId + '/' + serviceStateId).
                  success(function(data, status) {
                      if(DEBUG) console.log('Terminated %s', serviceStateId);
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Terminating instance failed", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          /*
           * Get the most recently retrieved host data.
           * This will also retrieve the data if it has not yet been
           * retrieved.
           *
           * @param {boolean} cacheOk Whether or not cached data is OK to use.
           * @param {function} callback Data passed to callback on success.
           */
          get_hosts: function(cacheOk, callback) {
              if (cacheOk && cached_hosts) {
                  if(DEBUG) console.log('Using cached hosts');
                  callback(cached_hosts);
              } else {
                  _get_hosts(callback);
              }
          },

          /*
           * Get a host
           * @param {string} hostID the host id
           * @param {function} callback host data is passed to a callback on success.
           */
          get_host: function(hostID, callback) {
              $http.noCacheGet('/hosts/' + hostID).
                  success(function(data, status) {
                      if(DEBUG) console.log('Retrieved %s for %s', data, hostID);
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Unable to acquire host", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          /*
           * Posts new host information to the server.
           *
           * @param {object} host New host details to be added.
           * @param {function} callback Add result passed to callback on success.
           */
          add_host: function(host, callback, errorCallback) {
              $http.post('/hosts/add', host).
                  success(function(data, status) {
                      $notification.create("", data.Detail).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }

                      if(errorCallback){
                          errorCallback(data);
                      }
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
                  success(function(data, status) {
                      $notification.create("Updated host", hostId).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Updating host failed", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }

                  });
          },

          /*
           * Deletes existing host.
           *
           * @param {string} hostId Unique identifier for host to be removed.
           * @param {function} callback Delete result passed to callback on success.
           */
          remove_host: function(hostId, callback) {
              $http.delete('/hosts/' + hostId).
                  success(function(data, status) {
                      $notification.create("Removed host", hostId).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Removing host failed", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          get_running_hosts: function(callback){
                $http.get("/hosts/running").success(function(data, status){
                    callback(data);
                }).error(function(data, status){
                  if(DEBUG) console.log('Unable to retrieve running hosts');
                  if (status === 401) {
                      unauthorized($location);
                  }
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

          /*
           * Get all defined services. Note that 2 arguments will be passed
           * to the callback function instead of the usual 1.
           *
           * The first argument to the callback is an array of all top level
           * services, with children attached.
           *
           * The second argument to the callback is a Map(Id -> Object) of all
           * services, with children attached.
           *
           * @param {boolean} cacheOk Whether or not cached data is OK to use.
           * @param {function} callback Executed on success.
           */
          get_services: function(cacheOk, callback) {
              if (cacheOk && cached_services && cached_services_map) {
                  if(DEBUG) console.log('Using cached services');
                  callback(cached_services, cached_services_map);
              } else {
                  _get_services_tree(callback);
              }
          },

          update_services: function(callback){
              var since,
                  url = "/services";

              if(cached_services_updated){
                  // calculate time in ms since last update
                  since = new Date().getTime() - cached_services_updated;
                  // add one second buffer to be sure we don't miss anything
                  since += 1000;

                  url += "?since=" + since;
              }

              // store the update time for comparison later
              cached_services_updated = new Date().getTime();

              // if services exist locally, update them
              if(cached_services && cached_services_map){
                  $http.get(url).then(function(data){
                      data.data.forEach(function(service){
                            var cachedService = cached_services_map[service.ID] || {};
                            // we can't just blow away the existing service
                            // because some controllers may be bound to it
                            // so update the fields on that service
                            for(var key in service){
                                if(DEBUG && cachedService[key] !== service[key] && typeof cachedService[key] !== "object"){
                                    console.log(key, "updated from", cachedService[key], "to", service[key]);
                                }
                                cachedService[key] = service[key];
                            }
                      });

                      callback(cached_services, cached_services_map);
                  });

              // we have gotten the initial list of services, so get em
              } else {
                 _get_services_tree(callback);
              }
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
                  success(function(data, status) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      if(DEBUG) console.log("Unable to retrieve service logs", data.Detail);
                      if (status === 401) {
                          unauthorized($location);
                      }
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
                  success(function(data, status) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      if(DEBUG) console.log("Unable to retrieve service logs", data.Detail);
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          /*
           * Determine if the user is logged into Docker Hub.
           * @param {function} callback boolean passed to callback on success.          
          */

          docker_is_logged_in: function(callback) {
            $http.noCacheGet('/dockerIsLoggedIn').
            success(function(data, status){
              callback(data.dockerLoggedIn);
            }).
            error(function(data, status) {
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
                  if(DEBUG) console.log('Using cached application templates');
                  callback(cached_app_templates);
              } else {
                  _get_app_templates(callback);
              }
          },

          add_app_template: function(fileData, callback){
              $.ajax( {
                  url: "/templates/add",
                  type: "POST",
                  data: fileData,
                  processData: false,
                  contentType: false,
                  success: function(data, status){
                      $notification.create("Added template", data.Detail).success();
                      callback(data);
                  },
                  error: function(data, status){
                      console.log(data);
                      $notification.create("Adding template failed", data.responseJSON.Detail).error();
                  }
              });
          },

          delete_app_template: function(templateID, callback){
              $http.delete('/templates/' + templateID).
                  success(function(data, status) {
                      $notification.create("Removed template", data.Detail).success();
                      callback(data);
                  }).
                  error(function(data, status){
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
              if(DEBUG) console.log('Adding detail: %s', JSON.stringify(service));
              $http.post('/services/add', service).
                  success(function(data, status) {
                      $notification.create("", "Added new service").success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Adding service failed", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          /*
           * Update an existing service
           *
           * @param {string} serviceId The ID of the service to update.
           * @param {object} editedService The modified service.
           * @param {function} callback Response passed to callback on success.
           */
          update_service: function(serviceId, editedService, success, fail) {
              fail = fail || angular.noop;

              $http.put('/services/' + serviceId, editedService).
                  success(function(data, status) {
                      $notification.create("Updated service", serviceId).success();
                      success(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Updating service failed", data.Detail).error();
                      fail(data, status);
                      if (status === 401) {
                          unauthorized($location);
                      }
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
                  success(function(data, status) {
                      $notification.create("", "Deployed application template").success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Deploying application template failed", data.Detail).error();
                      failCallback(data);
                      if (status === 401) {
                          unauthorized($location);
                      }
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
                  success(function(data, status) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Snapshot service failed", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
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
                  success(function(data, status) {
                      $notification.create("Removed service", serviceId).success();
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Removing service failed", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
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
                  success(function(data, status) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Was unable to start service", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
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
                  success(function(data, status) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Was unable to stop service", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
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
                  success(function(data, status) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("Was unable to restart service", data.Detail).error();
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },
          /**
           * Gets the Serviced version from the server
           */
          get_version: function(callback){
              $http.noCacheGet('/version').
                  success(function(data, status) {
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("", "Could not retrieve Serviced version from server.").warning();
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          /**
           * Creates a backup file of serviced
           */
          create_backup: function(success, fail){
              fail = fail || function(){};

              $http.noCacheGet('/backup/create').
                  success(function(data, status) {
                      success(data);
                  }).
                  error(function(data, status) {
                      if (status === 401) {
                          unauthorized($location);
                      }
                      fail(data, status);
                  });
          },

          /**
           * Restores a backup file of serviced
           */
          restore_backup: function(filename, success, fail){
              fail = fail || function(){};

              $http.get('/backup/restore?filename=' + filename).
                  success(function(data, status) {
                      success(data);
                  }).
                  error(function(data, status) {
                      if (status === 401) {
                          unauthorized($location);
                      }
                      fail(data, status);
                  });
          },

          get_backup_files: function(callback){
              $http.noCacheGet('/backup/list').
                  success(function(data, status) {
                      if(DEBUG) console.log('Retrieved list of backup files.');
                      callback(data);
                  }).
                  error(function(data, status) {
                      // TODO error screen
                      $notification.create("", "Failed retrieving list of backup files.").error();
                      if (status === 401) {
                          unauthorized($location);
                      }
                  });
          },

          get_backup_status: function(successCallback, failCallback){
              failCallback = failCallback || angular.noop;

              $http.noCacheGet('/backup/status').
                  success(function(data, status) {
                      if(DEBUG) console.log('Retrieved status of backup.');
                      successCallback(data);
                  }).
                  error(function(data, status) {
                      if (status === 401) {
                          unauthorized($location);
                      }
                      failCallback(data, status);
                  });
          },

          get_restore_status: function(successCallback, failCallback){
              failCallback = failCallback || angular.noop;

              $http.noCacheGet('/backup/restore/status').
                  success(function(data, status) {
                      if(DEBUG) console.log('Retrieved status of restore.');
                      successCallback(data);
                  }).
                  error(function(data, status) {
                      $notification.create("", 'Failed retrieving status of restore.').warning();
                      if (status === 401) {
                          unauthorized($location);
                      }
                      failCallback(data, status);
                  });
          },

          get_service_health: function(callback){

            var url = "/servicehealth";

            $http.get(url, { cache: healthcheckCache }).
              success(function(data, status) {
                  if(DEBUG) console.log('Retrieved health checks.');
                  callback(data);
              }).
              error(function(data, status) {
                  if(DEBUG) console.log('Failed retrieving health checks.');
                  if (status === 401) {
                      unauthorized($location);
                  }
              });

            
          },

          get_deployed_templates: function(deploymentDefinition, callback){
            $http.post('/templates/deploy/status', deploymentDefinition).
              success(function(data, status) {
                  if(DEBUG) console.log('Retrieved deployed template status.');
                  callback(data);
              });
          },

          get_active_templates: function(callback){
            $http.get('/templates/deploy/active', {cache: templatesCache}).
              success(function(data, status) {
                  if(DEBUG) console.log('Retrieved deployed template status.');
                  callback(data);
              });
          },

          get_stats: function(callback){
            $http.get("/stats").
              success(function(data, status) {
                  if(DEBUG) console.log('serviced is collecting stats.');
                  callback(status);
              }).
              error(function(data, status) {
                  // TODO error screen
                  $notification.create("", 'serviced is not collecting stats.').error();
                  if (status === 401) {
                      unauthorized($location);
                  }
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
