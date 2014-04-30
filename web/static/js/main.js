/*
 *******************************************************************************
 *
 *  Copyright (C) Zenoss, Inc. 2013, all rights reserved.
 *
 *  This content is made available according to terms specified in
 *  License.zenoss under the directory where your Zenoss product is installed.
 *
 *******************************************************************************
 */

/*******************************************************************************
 * Main module & controllers
 ******************************************************************************/
angular.module('controlplane', ['ngRoute', 'ngCookies','ngDragDrop','pascalprecht.translate', 'angularMoment']).
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
        }
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
        }
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

function ResourcesService($http, $location) {
    var cached_pools;
    var cached_hosts_for_pool = {};
    var cached_hosts;
    var cached_app_templates;
    var cached_services; // top level services only
    var cached_services_map; // map of services by by Id, with children attached

    var _get_services_tree = function(callback) {
        $http.get('/services').
            success(function(data, status) {
                console.log('Retrieved list of services');
                cached_services = [];
                cached_services_map = {};
                // Map by id
                data.map(function(svc) {
                    cached_services_map[svc.Id] = svc;
                });
                data.map(function(svc) {
                    if (svc.ParentServiceId !== '') {
                        var parent = cached_services_map[svc.ParentServiceId];
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
                console.error('Unable to retrieve services');
                if (status === 401) {
                    unauthorized($location);
                }
            });
    };

    var _get_app_templates = function(callback) {
        $http.get('/templates').
            success(function(data, status) {
                console.log('Retrieved list of app templates');
                cached_app_templates = data;
                callback(data);
            }).
            error(function(data, status) {
                // TODO error screen
                console.error('Unable to retrieve app templates');
                if (status === 401) {
                    unauthorized($location);
                }
            });
    };

    // Real implementation for acquiring list of resource pools
    var _get_pools = function(callback) {
        $http.get('/pools').
            success(function(data, status) {
                console.log('Retrieved list of pools');
                cached_pools = data
                callback(data);
            }).
            error(function(data, status) {
                // TODO error screen
                console.error('Unable to retrieve list of pools');
                if (status === 401) {
                    unauthorized($location);
                }
            });
    };

    var _get_hosts_for_pool = function(poolID, callback) {
        $http.get('/pools/' + poolID + '/hosts').
            success(function(data, status) {
                console.log('Retrieved hosts for pool %s', poolID);
                cached_hosts_for_pool[poolID] = data;
                callback(data);
            }).
            error(function(data, status) {
                // TODO error screen
                console.error('Unable to retrieve hosts for pool %s', poolID);
                if (status === 401) {
                    unauthorized($location);
                }
            });
    };

    var _get_hosts = function(callback) {
        $http.get('/hosts').
            success(function(data, status) {
                console.log('Retrieved host details');
                cached_hosts = data;
                callback(data);
            }).
            error(function(data, status) {
                // TODO error screen
                console.error('Unable to retrieve host details');
                if (status === 401) {
                    unauthorized($location);
                }
            });
    };

    return {
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
                console.log('Using cached pools');
                callback(cached_pools);
            } else {
                _get_pools(callback);
            }
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
                    console.log('Retrieved running services for %s', serviceId);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Unable to acquire running services: %s', JSON.stringify(data));
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
                    console.log('Retrieved list of virtual hosts');
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Unable to acquire virtual hosts: %s', JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        },

        /*
         * add a virtual host,
         */
        add_vhost: function(serviceId, application, virtualhost, callback) {
            var ep = '/services/' + serviceId + '/endpoint/' + application + '/vhosts/' + virtualhost
            var object = {'ServiceId':serviceId, 'Application':application, 'VirtualHostName':virtualhost};
            var payload = JSON.stringify( object);
            $http.put(ep, payload).
                success(function(data, status) {
                    console.log('Added virtual host: %s, %s', ep, JSON.stringify(data));
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Unable to add virtual hosts: %s, %s', ep, JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        },

        /*
         * Remove a virtual host
         */
        delete_vhost: function(serviceId, application, virtualhost, callback) {
            var ep = '/services/' + serviceId + '/endpoint/' + application + '/vhosts/' + virtualhost
            $http.delete(ep).
                success(function(data, status) {
                    console.log('Removed virtual host: %s, %s', ep, JSON.stringify(data));
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Unable to remove virtual hosts: %s, %s', ep, JSON.stringify(data));
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
                    console.log('Retrieved running services for %s', hostId);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Unable to acquire running services: %s', JSON.stringify(data));
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
                    console.error('Unable to acquire running services: %s', JSON.stringify(data));
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
            console.log('Adding detail: %s', JSON.stringify(pool));
            $http.post('/pools/add', pool).
                success(function(data, status) {
                    console.log('Added new pool');
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Adding pool failed: %s', JSON.stringify(data));
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
                    console.log('Updated pool %s', poolID);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Updating pool failed: %s', JSON.stringify(data));
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
                    console.log('Removed pool %s', poolID);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Removing pool failed: %s', JSON.stringify(data));
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
        add_pool_virtual_ip: function(pool, ip, callback) {
            var payload = JSON.stringify( {'poolID':pool,'VirtualIp':ip})
            console.log('Adding pool virtual ip: %s', payload);
            $http.put('/pools/' + pool + '/virtualip/' + ip, payload).
                success(function(data, status) {
                    console.log('Added new pool virtual ip');
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Adding pool virtual ip failed: %s', JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        },
        /*
         * Delete resource pool virtual ip
         *
         * @param {string} pool id to remove virtual ip
         * @param {string} ip virtual ip to remove from pool
         * @param {function} callback Add result passed to callback on success.
         */
        remove_pool_virtual_ip: function(pool, ip, callback) {
            console.log('Removing pool virtual ip: poolID:%s VirtualIp:%s', pool, ip);
            $http.delete('/pools/' + pool + '/virtualip/' + ip).
                success(function(data, status) {
                    console.log('Removed pool virtual ip');
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Remove pool virtual ip failed: %s', JSON.stringify(data));
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
                    console.log('Terminated %s', serviceStateId);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Terminating instance failed: %s', JSON.stringify(data));
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
                console.log('Using cached hosts');
                callback(cached_hosts);
            } else {
                _get_hosts(callback);
            }
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
                    console.log('Added new host');
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Adding host failed: %s', JSON.stringify(data));
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
                    console.log('Updated host %s', hostId);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Updating host failed: %s', JSON.stringify(data));
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
                    console.log('Removed host %s', hostId);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Removing host failed: %s', JSON.stringify(data));
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
                console.log('Using cached services');
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
                    console.error('Unable to retrieve service logs: %s', JSON.stringify(data));
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
                    console.error('Unable to retrieve service logs: %s', JSON.stringify(data));
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
                console.log('Using cached app templates');
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
            console.log('Adding detail: %s', JSON.stringify(service));
            $http.post('/services/add', service).
                success(function(data, status) {
                    console.log('Added new service');
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Adding service failed: %s', JSON.stringify(data));
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
                    console.log('Updated service %s', serviceId);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Updating service failed: %s', JSON.stringify(data));
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
        deploy_app_template: function(deployDef, callback) {
            $http.post('/templates/deploy', deployDef).
                success(function(data, status) {
                    console.log('Deployed app template');
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Deploying app template failed: %s', JSON.stringify(data));
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
                    console.error('Snapshot service failed: %s', JSON.stringify(data));
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
                    console.log('Removed service %s', serviceId);
                    callback(data);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('Removing service failed: %s', JSON.stringify(data));
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
                    console.error('Was unable to start service: %s', JSON.stringify(data));
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
                    console.error('Was unable to stop service: %s', JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        }
    };
}

function AuthService($cookies, $cookieStore, $location) {
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
            if ($cookies.ZCPToken !== undefined) {
                loggedIn = true;
                $scope.loggedIn = true;
                $scope.user = {
                    username: $cookies.ZUsername
                };
            } else {
                unauthorized($location);
            }
        },
    };
}

function StatsService($http, $location) {
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
                    console.log('serviced is collecting stats');
                    callback(status);
                }).
                error(function(data, status) {
                    // TODO error screen
                    console.error('serviced is not collecting stats');
                    callback(status);
                });
        }
    }
}

/*
 * Starting at some root node, recurse through children,
 * building a flattened array where each node has a depth
 * tracking field 'zendepth'.
 */
function flattenTree(depth, current) {
    // Exclude the root node
    var retVal = (depth === 0)? [] : [current];
    current.zendepth = depth;

    if (!current.children) {
        return retVal;
    }
    for (var i=0; i < current.children.length; i++) {
        retVal = retVal.concat(flattenTree(depth + 1, current.children[i]))
    }
    return retVal;
}

// return a url to a virtual host
function get_vhost_url( $location, vhost) {
  return $location.$$protocol + "://" + vhost + "." + $location.$$host + ":" + $location.$$port;
}

// collect all virtual hosts for provided service
function aggregateVhosts( service) {
  var vhosts = [];
  if (service.Endpoints) {
    for (var i in service.Endpoints) {
      var endpoint = service.Endpoints[i];
      if (endpoint.VHosts) {
        for ( var j in endpoint.VHosts) {
          var name = endpoint.VHosts[j];
          var vhost = {Name:name, Application:service.Name, ServiceEndpoint:endpoint.Application, ApplicationId:service.Id};
          vhosts.push( vhost)
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

// collect all virtual hosts options for provided service
function aggregateVhostOptions( service) {
  var options = [];
  if (service.Endpoints) {
    for (var i in service.Endpoints) {
      var endpoint = service.Endpoints[i];
      if (endpoint.VHosts) {
        var option = {
          ServiceId:service.Id,
          ServiceEndpoint:endpoint.Application,
          Value:service.Name + " - " + endpoint.Application
        };
        options.push( option);
      }
    }
  }

  for (var i in service.children) {
    var child_service = service.children[i];
    options = options.concat( aggregateVhostOptions( child_service));
  }

  return options;
}

function refreshServices($scope, servicesService, cacheOk, extraCallback) {
    // defend against empty scope
    if ($scope.services === undefined) {
        $scope.services = {};
    }
    console.log('refresh services called');
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
            updateRunning(svc);

            svc.deploymentClass = depClass;
            svc.deploymentIcon = iconClass;
        }

        if ($scope.params && $scope.params.serviceId) {
            $scope.services.current = $scope.services.mapped[$scope.params.serviceId];
            $scope.editService = $.extend({}, $scope.services.current);
            // we need a flattened view of all children

            if ($scope.services.current && $scope.services.current.children) {
                $scope.services.subservices = flattenTree(0, $scope.services.current);
                $scope.vhosts.data = aggregateVhosts( $scope.services.current);
                $scope.vhosts.options = aggregateVhostOptions( $scope.services.current);
                if ($scope.vhosts.options.length > 0) {
                  $scope.vhosts.add.app_ep = $scope.vhosts.options[0];
                }
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
    if (!mappedServices || !service.ParentServiceId || !mappedServices[service.ParentServiceId]) {
        return [ service ];
    }
    var lineage = getServiceLineage(mappedServices, mappedServices[service.ParentServiceId]);
    lineage.push(service);
    return lineage;
}

function refreshPools($scope, resourcesService, cachePools, extraCallback) {
    // defend against empty scope
    if ($scope.pools === undefined) {
        $scope.pools = {};
    }
    console.log('Refreshing pools');
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

function toggleRunning(app, status, servicesService) {
    var newState = -1;
    switch(status) {
        case 'start': newState = 1; break;
        case 'stop': newState = 0; break;
        case 'restart': newState = -1; break;
    }
    if (newState === app.DesiredState) {
        console.log('Same status. Ignoring click');
        return;
    }

    // recursively updates the text of the status button, this
    // is so that when stopping takes a long time you can see that
    // something is happening. This doesn't update the color
    function updateAppText(app, text, notRunningText) {
        var i;
        app.runningText = text;
        app.notRunningText = notRunningText;
        if (!app.children) {
            return;
        }
        for (i=0; i<app.children.length;i++) {
            updateAppText(app.children[i], text, notRunningText);
        }
    }

    // updates the color and the running/non-running text of the
    // status buttons
    function updateApp(app, desiredState) {
        var i, child;
        updateRunning(app);
        if (app.children && app.children.length) {
            for (i=0; i<app.children.length;i++) {
                child = app.children[i];
                child.DesiredState = desiredState;
                updateRunning(child);
                if (child.children && child.children.length) {
                    updateApp(child, desiredState);
                }
            }
        }
    }
    // stop service
    if ((newState == 0) || (newState == -1)) {
        app.DesiredState = newState;
        servicesService.stop_service(app.Id, function() {
            updateApp(app, newState);
        });
        updateAppText(app, "stopping...", "ctl_running_blank");
    }

    // start service
    if ((newState == 1) || (newState == -1)) {
        app.DesiredState = newState;
        servicesService.start_service(app.Id, function() {
            updateApp(app, newState);
        });
        updateAppText(app, "ctl_running_blank", "starting...");
    }
}

function updateRunning(app) {
    if (app.DesiredState === 1) {
        app.runningText = "ctl_running_started";
        app.notRunningText = "ctl_running_blank"; // &nbsp
        app.runningClass = "btn btn-success active";
        app.notRunningClass = "btn btn-default off";
    } else if (app.DesiredState === -1) {
        app.runningText = "ctl_running_restarting";
        app.notRunningText = "ctl_running_blank"; // &nbsp
        app.runningClass = "btn btn-info active";
        app.notRunningClass = "btn btn-default off";
    } else {
        app.runningText = "ctl_running_blank"; // &nbsp
        app.notRunningText = "ctl_running_stopped";
        app.runningClass = "btn btn-default off";
        app.notRunningClass = "btn btn-danger active";
    }
    if (app.Deployment !== "successful") {
        app.runningClass += " disabled";
        app.notRunningClass += " disabled";
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
            updateRunning(runningServices[i]);
        }
    });
}

function refreshRunningForService($scope, resourcesService, serviceId, extracallback) {
    if ($scope.running === undefined) {
        $scope.running = {};
    }

    resourcesService.get_running_services_for_service(serviceId, function(runningServices) {
        $scope.running.data = runningServices;
        $scope.running.sort = 'InstanceId';
        for (var i=0; i < runningServices.length; i++) {
            runningServices[i].DesiredState = 1; // All should be running
            runningServices[i].Deployment = 'successful'; // TODO: Replace
            updateRunning(runningServices[i]);
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
        console.log('Unable to update host pool paths');
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

function unauthorized($location) {
    console.log('You don\'t appear to be logged in.');
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
    console.log('Resetting ' + table.sort + ' to down.');
    table.sort_icons[table.sort] = 'glyphicon-chevron-down';

    if (table.sort === order) {
        table.sort = "-" + order;
        table.sort_icons[table.sort] = 'glyphicon-chevron-down';
        console.log('Sorting by -' + order);
    } else {
        table.sort = order;
        table.sort_icons[table.sort] = 'glyphicon-chevron-up';
        console.log('Sorting by ' + order);
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
};

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

function removePool(scope, poolID){
    // clear out the pool we just deleted in case it is stuck in a database index
    for(var i=0; i < scope.pools.data.length; ++i){
        if(scope.pools.data[i].ID === poolID){
            scope.pools.data.splice(i, 1);
        }
    }
    for(var i=0; i < scope.pools.flattened.length; ++i){
        if(scope.pools.flattened[i].ID === poolID){
            scope.pools.flattened.splice(i, 1);
        }
    }
    for(var i=0; i < scope.pools.tree.length; ++i){
        if(scope.pools.tree[i].ID === poolID){
            scope.pools.tree.splice(i, 1);
        }
    }
}
