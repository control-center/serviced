// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

/*******************************************************************************
 * Main module & controllers
 ******************************************************************************/
angular.module('controlplane', ['ngRoute', 'ngCookies','ngDragDrop','pascalprecht.translate', 'angularMoment', 'zenNotify', 'serviceHealth', 'modalService']).
    config(['$routeProvider', function($routeProvider) {
        $routeProvider.
            when('/entry', {
                templateUrl: '/static/partials/main.html',
                controller: EntryControl}).
            when('/login', {
                templateUrl: '/static/partials/login.html',
                controller: LoginControl}).
            when('/logs', {
                templateUrl: '/static/partials/logs.html',
                controller: LogControl}).
            when('/services/:serviceId', {
                templateUrl: '/static/partials/view-subservices.html',
                controller: SubServiceControl}).
            when('/apps', {
                templateUrl: '/static/partials/view-apps.html',
                controller: DeployedAppsControl}).
            when('/hosts', {
                templateUrl: '/static/partials/view-hosts.html',
                controller: HostsControl}).
            when('/hostsmap', {
                templateUrl: '/static/partials/view-host-map.html',
                controller: HostsMapControl}).
            when('/servicesmap', {
                templateUrl: '/static/partials/view-service-map.html',
                controller: ServicesMapControl}).
            when('/hosts/:hostId', {
                templateUrl: '/static/partials/view-host-details.html',
                controller: HostDetailsControl}).
            when('/jobs', {
                templateUrl: '/static/partials/celery-log.html',
                controller: CeleryLogControl}).
            when('/pools', {
                templateUrl: '/static/partials/view-pools.html',
                controller: PoolsControl}).
            when('/pools/:poolID', {
                templateUrl: '/static/partials/view-pool-details.html',
                controller: PoolDetailsControl}).
            when('/devmode', {
                templateUrl: '/static/partials/view-devmode.html',
                controller: DevControl
            }).
            when('/backuprestore', {
                templateUrl: '/static/partials/view-backuprestore.html',
                controller: BackupRestoreControl
            }).
            when('/isvcs', {
                templateUrl: '/static/partials/view-isvcs.html',
                controller: IsvcsControl
            }).
            otherwise({redirectTo: '/entry'});
    }]).
    config(['$translateProvider', function($translateProvider) {

        $translateProvider.useStaticFilesLoader({
            prefix: '/static/i18n/',
            suffix: '.json'
        });
        $translateProvider.preferredLanguage('en_US');
    }]).
    /**
     * This is a fix for https://jira.zenoss.com/browse/ZEN-10263
     * It makes sure that inputs that are filled in by autofill (like when the browser remembers the password)
     * are updated in the $scope. See the partials/login.html for an example
     **/
    directive('formAutofillFix', function() {
        return function(scope, elem, attrs) {
            // Fixes Chrome bug: https://groups.google.com/forum/#!topic/angular/6NlucSskQjY
            elem.prop('method', 'POST');

            // Fix autofill issues where Angular doesn't know about autofilled inputs
            if(attrs.ngSubmit) {
                window.setTimeout(function() {
                    elem.unbind('submit').submit(function(e) {
                        e.preventDefault();
                        elem.find('input, textarea, select').trigger('input').trigger('change').trigger('keydown');
                        scope.$apply(attrs.ngSubmit);
                    });
                }, 0);
            }
        };
    }).
    directive('showIfEmpty', function(){
        return function(scope, elem, attrs){
            scope.showIfEmpty();
        };
    }).
    directive('popover', function(){
        return function(scope, elem, attrs){
            $(elem).popover({
                title: attrs.popoverTitle,
                trigger: "hover",
                html: true,
                content: attrs.popover
            });
        };
    }).
    factory('resourcesService', ResourcesService).
    factory('authService', AuthService).
    factory('statsService', StatsService).
    filter('treeFilter', function() {
        /*
         * @param items The array from ng-repeat
         * @param field Field on items to check for validity
         * @param validData Object with allowed objects
         */
        return function(items, field, validData) {
            if (!validData) {
                return items;
            }
            return items.filter(function(elem) {
                return validData[elem[field]] != null;
            });
        };
    }).
    filter('page', function() {
        return function(items, hosts) {
            if (!items) return;

            var pageSize = hosts.pageSize? hosts.pageSize : 5;
            hosts.pages = Math.max(1, Math.ceil(items.length / pageSize));
            if (!hosts.page || hosts.page >= hosts.pages) {
                hosts.page = 0;
            }
            var page = hosts.page? hosts.page : 0;

            var start = page * pageSize;
            return items.splice(start, pageSize);
        };
    }).
    filter('toGB', function(){
        return function(input){
            return (input/1073741824).toFixed(2) + " GB";
        };
    }).
    directive('scroll', function($rootScope, $window, $timeout) {
        return {
            link: function(scope, elem, attr) {
                $window = angular.element($window);
                var handler = function() {
                    var winEdge, elEdge, dataHidden, scroll;
                    winEdge = $window.height() + $window.scrollTop();
                    elEdge = elem.offset().top + elem.height();
                    dataHidden = elEdge - winEdge;
                    if (dataHidden < parseInt(attr.scrollSize, 10)) {
                        if ($rootScope.$$phase) {
                            if (scope.$eval(attr.scroll)) {
                                $timeout(handler, 100);
                            }
                        } else {
                            if (scope.$apply(attr.scroll)) {
                                $timeout(handler, 100);
                            }
                        }
                    }
                };
                if (attr.scrollHandlerObj && attr.scrollHandlerField) {
                    var obj = scope[attr.scrollHandlerObj];
                    obj[attr.scrollHandlerField] = handler;
                }
                $window.on('scroll', handler);
                $window.on('resize', handler);
                scope.$on('$destroy', function() {
                    $window.off('scroll', handler);
                    $window.off('resize', handler);
                    return true;
                });
                return $timeout((function() {
                    return handler();
                }), 100);
            }
        };
    });

/* begin constants */
var POOL_ICON_CLOSED = 'glyphicon glyphicon-play btn-link';
var POOL_ICON_OPEN = 'glyphicon glyphicon-play rotate-down btn-link';
var POOL_CHILDREN_CLOSED = 'hidden';
var POOL_CHILDREN_OPEN = 'nav-tree';
/* end constants */

// set verbosity of console.logs
var DEBUG = false;

function addChildren(allowed, parent) {
    allowed[parent.Id] = true;
    if (parent.children) {
        for (var i=0; i < parent.children.length; i++) {
            addChildren(allowed, parent.children[i]);
        }
    }
}

function updateLanguage($scope, $cookies, $translate) {
    var ln = 'en_US';
    if ($cookies.Language === undefined) {

    } else {
        ln = $cookies.Language;
    }
    if ($scope.user) {
        $scope.user.language = ln;
    }
    $translate.uses(ln);
}

function ResourcesService($http, $location, $notification) {
    var cached_pools;
    var cached_hosts_for_pool = {};
    var cached_hosts;
    var cached_app_templates;
    var cached_services; // top level services only
    var cached_services_map; // map of services by by Id, with children attached

    var _get_services_tree = function(callback) {
        $http.get('/services').
            success(function(data, status) {
                if(DEBUG) console.log('Retrieved list of services');
                cached_services = [];
                cached_services_map = {};
                // Map by id
                data.map(function(svc) {
                    cached_services_map[svc.ID] = svc;
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
                $notification.create("",'Unable to retrieve services').error();
                if (status === 401) {
                    unauthorized($location);
                }
            });
    };

    var _get_app_templates = function(callback) {
        $http.get('/templates').
            success(function(data, status) {
                if(DEBUG) console.log('Retrieved list of app templates');
                cached_app_templates = data;
                callback(data);
            }).
            error(function(data, status) {
                // TODO error screen
                $notification.create("",'Unable to retrieve app templates').error();
                if (status === 401) {
                    unauthorized($location);
                }
            });
    };

    // Real implementation for acquiring list of resource pools
    var _get_pools = function(callback) {
        $http.get('/pools').
            success(function(data, status) {
                if(DEBUG) console.log('Retrieved list of pools');
                cached_pools = data;
                callback(data);
            }).
            error(function(data, status) {
                // TODO error screen
                $notification.create("",'Unable to retrieve list of pools').error();
                if (status === 401) {
                    unauthorized($location);
                }
            });
    };

    var _get_hosts_for_pool = function(poolID, callback) {
        $http.get('/pools/' + poolID + '/hosts').
            success(function(data, status) {
                if(DEBUG) console.log('Retrieved hosts for pool %s', poolID);
                cached_hosts_for_pool[poolID] = data;
                callback(data);
            }).
            error(function(data, status) {
                // TODO error screen
                $notification.create("",('Unable to retrieve hosts for pool ' + poolID)).error();
                if (status === 401) {
                    unauthorized($location);
                }
            });
    };

    var _get_hosts = function(callback) {
        $http.get('/hosts').
            success(function(data, status) {
                if(DEBUG) console.log('Retrieved host details');
                cached_hosts = data;
                callback(data);
            }).
            error(function(data, status) {
                // TODO error screen
                $notification.create("",('Unable to retrieve host details')).error();
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
          if (ip != null) {
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
            $http.get('/pools/' + poolID).
                success(function(data, status) {
                    if(DEBUG) console.log('Retrieved %s for %s', data, poolID);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("Unable to acquire pool", data.Detail).error();
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
            $http.get('/pools/' + poolID + "/ips").
                success(function(data, status) {
                    if(DEBUG) console.log('Retrieved %s for %s', data, poolID);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("Unable to acquire pool", data.Detail).error();
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
            $http.get('/services/' + serviceId + '/running').
                success(function(data, status) {
                    if(DEBUG) console.log('Retrieved running services for %s', serviceId);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("Unable to acquire running services", data.Detail).error();
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
            $http.get('/vhosts').
                success(function(data, status) {
                    if(DEBUG) console.log('Retrieved list of virtual hosts');
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("Unable to acquire virtual hosts", data.Detail).error();
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
            $http.get('/hosts/' + hostId + '/running').
                success(function(data, status) {
                    if(DEBUG) console.log('Retrieved running services for %s', hostId);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("Unable to acquire running services", data.Detail).error();
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
            $http.get('/running').
                success(function(data, status) {
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("Unable to acquire running services", data.Detail).error();
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
            $http.get('/hosts/' + hostID).
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
        add_host: function(host, callback) {
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

        /*
         * Retrieve some (probably not the one you want) set of logs for a
         * defined service. To get more specific logs, use
         * get_service_state_logs.
         *
         * @param {string} serviceId ID of the service to retrieve logs for.
         * @param {function} callback Log data passed to callback on success.
         */
        get_service_logs: function(serviceId, callback) {
            $http.get('/services/' + serviceId + '/logs').
                success(function(data, status) {
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("Unable to retrieve service logs", data.Detail).error();
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
            $http.get('/services/' + serviceId + '/' + serviceStateId + '/logs').
                success(function(data, status) {
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("Unable to retrieve service logs", data.Detail).error();
                    if (status === 401) {
                        unauthorized($location);
                    }
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
                if(DEBUG) console.log('Using cached app templates');
                callback(cached_app_templates);
            } else {
                _get_app_templates(callback);
            }
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
        update_service: function(serviceId, editedService, callback) {
            $http.put('/services/' + serviceId, editedService).
                success(function(data, status) {
                    $notification.create("Updated service", serviceId).success();
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("Updating service failed", data.Detail).error();
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
                    $notification.create("", "Deployed app template").success();
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("Deploying app template failed", data.Detail).error();
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
            $http.get('/services/' + serviceId + '/snapshot').
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
         */
        start_service: function(serviceId, callback) {
            $http.put('/services/' + serviceId + '/startService').
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
        stop_service: function(serviceId, callback) {
            $http.put('/services/' + serviceId + '/stopService').
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
        /**
         * Gets the Serviced version from the server
         */
        get_version: function(callback){
            $http.get('/version').
                success(function(data, status) {
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("", "Could not retrieve Serviced version from server.").error();
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        },

        /**
         * Creates a backup file of serviced
         */
        create_backup: function(callback){
            $http.get('/backup/create').
                success(function(data, status) {
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

        /**
         * Restores a backup file of serviced
         */
        restore_backup: function(filename, callback){
            $http.get('/backup/restore?filename=' + filename).
                success(function(data, status) {
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

        get_backup_files: function(callback){
            $http.get('/backup/list').
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

            $http({url: '/backup/status', method: "GET", params: {'time': new Date().getTime()}}).
                success(function(data, status) {
                    if(DEBUG) console.log('Retrieved status of backup.');
                    successCallback(data);
                }).
                error(function(data, status) {
                    $notification.create("", 'Failed retrieving status of backup.').error();
                    if (status === 401) {
                        unauthorized($location);
                    }
                    failCallback(data, status);
                });
        },

        get_restore_status: function(successCallback, failCallback){
            failCallback = failCallback || angular.noop;

            $http({url: '/backup/restore/status', method: "GET", params: {'time': new Date().getTime()}}).
                success(function(data, status) {
                    if(DEBUG) console.log('Retrieved status of restore.');
                    successCallback(data);
                }).
                error(function(data, status) {
                    $notification.create("", 'Failed retrieving status of restore.').error();
                    if (status === 401) {
                        unauthorized($location);
                    }
                    failCallback(data, status);
                });
        }
    };
}

function AuthService($cookies, $cookieStore, $location, $notification) {
    var loggedIn = false;
    var userName = null;
    return {

        /*
         * Called when we have positively determined that a user is or is not
         * logged in.
         *
         * @param {boolean} truth Whether the user is logged in.
         */
        login: function(truth, username) {
            loggedIn = truth;
            userName = username;
        },

        /*
         * Check whether the user appears to be logged in. Update path if not.
         *
         * @param {object} scope The 'loggedIn' property will be set if true
         */
        checkLogin: function($scope) {
            $scope.dev = $cookieStore.get('ZDevMode');
            if (loggedIn) {
                $scope.loggedIn = true;
                $scope.user = {
                    username: $cookies.ZUsername
                };
                return;
            }
            if ($cookies.ZCPToken) {
                loggedIn = true;
                $scope.loggedIn = true;
                $scope.user = {
                    username: $cookies.ZUsername
                };
            } else {
                unauthorized($location);
            }
        }
    };
}

function StatsService($http, $location, $notification) {
    return {
        /*
         * Get the list of services currently running on a particular host.
         *
         * @param {string} hostId The ID of the host to retrieve running services for.
         * @param {function} callback Running services are passed to callback on success.
         */
        is_collecting: function(callback) {
            $http.get('/stats').
                success(function(data, status) {
                    if(DEBUG) console.log('serviced is collecting stats');
                    callback(status);
                }).
                error(function(data, status) {
                    // TODO error screen
                    $notification.create("", 'serviced is not collecting stats').error();
                    callback(status);
                });
        }
    };
}

/*
 * Starting at some root node, recurse through children,
 * building a flattened array where each node has a depth
 * tracking field 'zendepth'.
 */
function flattenTree(depth, current, sortFunction) {
    // Exclude the root node
    var retVal = (depth === 0)? [] : [current];
    current.zendepth = depth;

    if (!current.children) {
        return retVal;
    }
    if (sortFunction !== undefined) {
        current.children.sort(sortFunction);
    }
    for (var i=0; i < current.children.length; i++) {
        retVal = retVal.concat(flattenTree(depth + 1, current.children[i]));
    }
    return retVal;
}

// return a url to a virtual host
function get_vhost_url( $location, vhost) {
  return $location.$$protocol + "://" + vhost + "." + $location.$$host + ":" + $location.$$port;
}

// collect all virtual hosts for provided service
function aggregateVhosts(service) {
  var vhosts = [];
  if (service.Endpoints) {
    for (var i in service.Endpoints) {
      var endpoint = service.Endpoints[i];
      if (endpoint.VHosts) {
        for ( var j in endpoint.VHosts) {
          var name = endpoint.VHosts[j];
          var vhost = {Name:name, Application:service.Name, ServiceEndpoint:endpoint.Application, ApplicationId:service.ID};
          vhosts.push( vhost);
        }
      }
    }
  }
  for (var i in service.children) {
    var child_service = service.children[i];
    vhosts = vhosts.concat( aggregateVhosts( child_service));
  }
  return vhosts;
}
// collect all address assignments for a service
function aggregateAddressAssigments(service, api) {
  var assignments = [];
  if (service.Endpoints) {
    for (var i in service.Endpoints) {
      var endpoint = service.Endpoints[i];
      if (endpoint.AddressConfig.Port > 0 && endpoint.AddressConfig.Protocol != "") {
        var assignment = {
          'ID': endpoint.AddressAssignment.ID,
          'AssignmentType': endpoint.AddressAssignment.AssignmentType,
          'EndpointName': endpoint.AddressAssignment.EndpointName,
          'HostID': endpoint.AddressAssignment.HostID,
          'HostName': 'unknown',
          'PoolID': endpoint.AddressAssignment.PoolID,
          'IPAddr': endpoint.AddressAssignment.IPAddr,
          'Port': endpoint.AddressConfig.Port,
          'ServiceID': service.ID,
          'ServiceName': service.Name
        };
        if (assignment.HostID !== "") {
          api.get_host( assignment.HostID, function(data) {
            assignment.HostName = data.Name;
          })
        }
        assignments.push( assignment);
      }
    }
  }

  for (var i in service.children) {
    var child_service = service.children[i];
    assignments = assignments.concat( aggregateAddressAssigments( child_service, api));
  }
  return assignments;
}

// collect all virtual hosts options for provided service
function aggregateVhostOptions(service) {
  var options = [];
  if (service.Endpoints) {
    for (var i in service.Endpoints) {
      var endpoint = service.Endpoints[i];
      if (endpoint.Purpose == "export") {
        var option = {
          ServiceID:service.ID,
          ServiceEndpoint:endpoint.Application,
          Value:service.Name + " - " + endpoint.Application
        };
        options.push(option);
      }
    }
  }

  for (var i in service.children) {
    var child_service = service.children[i];
    options = options.concat(aggregateVhostOptions(child_service));
  }

  return options;
}

function refreshServices($scope, servicesService, cacheOk, extraCallback) {
    // defend against empty scope
    if ($scope.services === undefined) {
        $scope.services = {};
    }
    if(DEBUG) console.log('refresh services called');
    servicesService.get_services(cacheOk, function(topServices, mappedServices) {
        $scope.services.data = topServices;
        $scope.services.mapped = mappedServices;

        for (var key in $scope.services.mapped) {
            var svc = $scope.services.mapped[key];
            var depClass = "";
            var iconClass = "";
            var runningClass = "";
            var notRunningClass = "";
            svc.Deployment = 'successful'; // TODO: replace with real data

            switch(svc.Deployment) {
            case "successful":
                depClass = "deploy-success";
                iconClass = "glyphicon glyphicon-ok";
                break;
            case "failed":
                depClass = "deploy-error";
                iconClass = "glyphicon glyphicon-remove";
                break;
            case "in-process":
                depClass = "deploy-info";
                iconClass = "glyphicon glyphicon-refresh";
                break;
            default:
                depClass = "deploy-warning";
                iconClass = "glyphicon glyphicon-question-sign";
                break;
            }

            svc.deploymentClass = depClass;
            svc.deploymentIcon = iconClass;
        }

        if ($scope.params && $scope.params.serviceId) {
            $scope.services.current = $scope.services.mapped[$scope.params.serviceId];
            $scope.editService = $.extend({}, $scope.services.current);

            // we need a flattened view of all children
            if ($scope.services.current && $scope.services.current.children) {
                $scope.services.subservices = flattenTree(0, $scope.services.current, function(a, b) {
                    return a.Name.toLowerCase() < b.Name.toLowerCase() ? -1 : 1;
                });
            }

            // aggregate virtual ip and virtual host data
            if ($scope.services.current) {
                $scope.vhosts.data = aggregateVhosts( $scope.services.current);
                $scope.vhosts.options = aggregateVhostOptions( $scope.services.current);
                if ($scope.vhosts.options.length > 0) {
                  $scope.vhosts.add.app_ep = $scope.vhosts.options[0];
                }
                $scope.ips.data = aggregateAddressAssigments($scope.services.current, servicesService);
            }
        }
        if (extraCallback) {
            extraCallback();
        }
    });
}

function getFullPath(allPools, pool) {
    if (!allPools || !pool.ParentId || !allPools[pool.ParentId]) {
        return pool.ID;
    }
    return getFullPath(allPools, allPools[pool.ParentId]) + " > " + pool.ID;
}

function getServiceLineage(mappedServices, service) {
    if (!mappedServices || !service.ParentServiceID || !mappedServices[service.ParentServiceID]) {
        return [ service ];
    }
    var lineage = getServiceLineage(mappedServices, mappedServices[service.ParentServiceID]);
    lineage.push(service);
    return lineage;
}

function refreshPools($scope, resourcesService, cachePools, extraCallback) {
    // defend against empty scope
    if ($scope.pools === undefined) {
        $scope.pools = {};
    }
    if(DEBUG) console.log('Refreshing pools');
    resourcesService.get_pools(cachePools, function(allPools) {
        $scope.pools.mapped = allPools;
        $scope.pools.data = map_to_array(allPools);
        $scope.pools.tree = [];

        var flatroot = { children: [] };
        for (var key in allPools) {
            var p = allPools[key];
            p.collapsed = false;
            p.childrenClass = 'nav-tree';
            p.dropped = [];
            p.itemClass = 'pool-data';
            if (p.icon === undefined) {
                p.icon = 'glyphicon spacer disabled';
            }
            var parent = allPools[p.ParentId];
            if (parent) {
                if (parent.children === undefined) {
                    parent.children = [];
                    parent.icon = POOL_ICON_OPEN;
                }
                parent.children.push(p);
                p.fullPath = getFullPath(allPools, p);
            } else {
                flatroot.children.push(p);
                $scope.pools.tree.push(p);
                p.fullPath = p.ID;
            }
        }

        if ($scope.params && $scope.params.poolID) {
            $scope.pools.current = allPools[$scope.params.poolID];
            $scope.editPool = $.extend({}, $scope.pools.current);
        }

        $scope.pools.flattened = flattenTree(0, flatroot);

        if (extraCallback) {
            extraCallback();
        }
    });
}

function toggleRunning(app, status, servicesService, serviceId) {
    serviceId = serviceId || app.ID;

    var newState = -1;
    switch(status) {
        case 'start': newState = 1; break;
        case 'stop': newState = 0; break;
        case 'restart': newState = -1; break;
    }
    if (newState === app.DesiredState) {
        if(DEBUG) console.log('Same status. Ignoring click');
        return;
    }

    // recursively set service's children to its desired state
    function updateApp(app) {
        var i, child;
        if (app.children && app.children.length) {
            for (i=0; i<app.children.length;i++) {
                app.children[i].DesiredState = app.DesiredState;
                updateApp(app.children[i]);
            }
        }
    }

    // stop service
    if ((newState === 0) || (newState === -1)) {
        app.DesiredState = newState;
        servicesService.stop_service(serviceId, function() {
            updateApp(app);
        });
    }

    // start service
    if ((newState === 1) || (newState === -1)) {
        app.DesiredState = newState;
        servicesService.start_service(serviceId, function() {
            updateApp(app);
        });
    }
}

function refreshHosts($scope, resourcesService, cacheHosts, extraCallback) {
    // defend against empty scope
    if ($scope.hosts === undefined) {
        $scope.hosts = {};
    }

    resourcesService.get_hosts(cacheHosts, function(allHosts) {
        // This is a Map(Id -> Host)
        $scope.hosts.mapped = allHosts;

        // Get array of all hosts
        $scope.hosts.all = map_to_array(allHosts);

        // This method gets called more than once. We don't watch to keep
        // setting watches if we've already got one.
        if ($scope.pools === undefined || $scope.pools.mapped === undefined) {
            // Transfer path from pool to host
            $scope.$watch('pools.mapped', function() {
                fix_pool_paths($scope);
            });
        } else {
            fix_pool_paths($scope);
        }

        if ($scope.params && $scope.params.hostId) {
            var current = allHosts[$scope.params.hostId];
            if (current) {
                $scope.editHost = $.extend({}, current);
                $scope.hosts.current = current;
            }
        }

        if (extraCallback) {
            extraCallback();
        }
    });
}

function refreshRunningForHost($scope, resourcesService, hostId) {
    if ($scope.running === undefined) {
        $scope.running = {};
    }

    resourcesService.get_running_services_for_host(hostId, function(runningServices) {
        $scope.running.data = runningServices;
        for (var i=0; i < runningServices.length; i++) {
            runningServices[i].DesiredState = 1; // All should be running
            runningServices[i].Deployment = 'successful'; // TODO: Replace
        }
    });
}

function refreshRunningForService($scope, resourcesService, serviceId, extracallback) {
    if ($scope.running === undefined) {
        $scope.running = {};
    }

    resourcesService.get_running_services_for_service(serviceId, function(runningServices) {
        $scope.running.data = runningServices;
        $scope.running.sort = 'InstanceID';
        for (var i=0; i < runningServices.length; i++) {
            runningServices[i].DesiredState = 1; // All should be running
            runningServices[i].Deployment = 'successful'; // TODO: Replace
        }

        if (extracallback) {
            extracallback();
        }
    });
}

function fix_pool_paths($scope) {
    if ($scope.pools && $scope.pools.mapped && $scope.hosts && $scope.hosts.all) {
        for(var i=0; i < $scope.hosts.all.length; i++) {
            var host = $scope.hosts.all[i];
            host.fullPath = $scope.pools.mapped[host.PoolID].fullPath;
        }
    } else {
        console.error('Unable to update host pool paths');
    }
}

/*
 * Helper function transforms Map(K -> V) into Array(V)
 */
function map_to_array(data) {
    var arr = [];
    for (var key in data) {
        arr[arr.length] = data[key];
    }
    return arr;
}

function unauthorized($location, $notification) {
    console.error('You don\'t appear to be logged in.');
    $location.path('/login');
}

/*
 * Helper function acquires next URL from data that looks like this:
 *
   {
     ...,
     Links: [ { Name: 'Next', Url: '/some/url' }, ... ]
   }
 *
 */
function next_url(data) {
    return data.Links.filter(function(e) {
        return e.Name == 'Next';
    })[0].Url;
}

function set_order(order, table) {
    // Reset the icon for the last order
    if(DEBUG) console.log('Resetting ' + table.sort + ' to down.');
    table.sort_icons[table.sort] = 'glyphicon-chevron-down';

    if (table.sort === order) {
        table.sort = "-" + order;
        table.sort_icons[table.sort] = 'glyphicon-chevron-down';
        if(DEBUG) console.log('Sorting by -' + order);
    } else {
        table.sort = order;
        table.sort_icons[table.sort] = 'glyphicon-chevron-up';
        if(DEBUG) console.log('Sorting ' + table +' by ' + order);
    }
}

function get_order_class(order, table) {
    return 'glyphicon btn-link sort pull-right ' + table.sort_icons[order] +
        ((table.sort === order || table.sort === '-' + order) ? ' active' : '');
}

function buildTable(sort, headers) {
    var sort_icons = {};
    for(var i=0; i < headers.length; i++) {
        sort_icons[headers[i].id] = (sort === headers[i].id?
            'glyphicon-chevron-up' : 'glyphicon-chevron-down');
    }

    return {
        sort: sort,
        headers: headers,
        sort_icons: sort_icons,
        set_order: set_order,
        get_order_class: get_order_class,
        page: 1,
        pageSize: 5
    };
}

function indentClass(depth) {
    return 'indent' + (depth -1);
}

function toggleCollapse(child, collapsed) {
    child.parentCollapsed = collapsed;
    // We're done if this node does not have any children OR if this node is
    // already collapsed
    if (!child.children || child.collapsed) {
        return;
    }
    // Mark all children as having a collapsed parent
    for(var i=0; i < child.children.length; i++) {
        toggleCollapse(child.children[i], collapsed);
    }
}

function itemClass(item) {
    var cls = item.current? 'current' : '';
    if (item.parentCollapsed) {
        cls += ' hidden';
    }
    return cls;
}

// keep notifications stuck to bottom of nav, or top of window
// if nav is out ovf view.
var $window = $(window);
$window.on("scroll", function(){
    var currScrollTop = $window.scrollTop(),
        $notifications = $("#notifications");

    if(currScrollTop > 0){
        var top = Math.max(72 - currScrollTop, 0);
        $notifications.css("top", top+"px");
    }else{
        $notifications.css("top", "72px");
    }
});
