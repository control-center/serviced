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
angular.module('controlplane', ['ngCookies']).
    config(['$routeProvider', function($routeProvider) {
        $routeProvider.
            when('/entry', { 
                templateUrl: '/static/partials/main.html',
                controller: EntryControl}).
            when('/services', {
                templateUrl: '/static/partials/services.html',
                controller: ConfigurationControl}).
            when('/resources', {
                templateUrl: '/static/partials/resources.html',
                controller: ResourcesControl}).
            when('/pools/:poolId', {
                templateUrl: '/static/partials/pool-details.html',
                controller: PoolControl}).
            when('/pools/:poolId/hosts/:hostId', {
                templateUrl: '/static/partials/host-details.html',
                controller: HostControl}).
            when('/wizard/start', {
                templateUrl: '/static/partials/wizard_splash.html', 
                controller: WizardControl}).
            when('/wizard/page1', {
                templateUrl: '/static/partials/wizard1.html', 
                controller: WizardControl}).
            when('/wizard/page2', {
                templateUrl: '/static/partials/wizard2.html', 
                controller: WizardControl}).
            when('/wizard/finish', {
                templateUrl: '/static/partials/wizard_finish.html', 
                controller: WizardControl}).
            otherwise({redirectTo: '/entry'});
    }]).
    factory('resourcesService', ResourcesService).
    factory('wizardService', WizardService).
    factory('servicesService', ServicesService);

function EntryControl($scope) {
    $scope.mainlinks = [
        { url: '#/wizard/start', label: 'Install' },
        { url: '#/services', label: 'Configure services' },
        { url: '#/resources', label: 'Manage resources' }
    ];
}

function LoginControl($scope, $http, $location) {
    $scope.brand_label = "SERVICE DYNAMICS";
    $scope.login_button_text = "Log In";
    $scope.login_alert = $('#login_alert')
    $scope.login_alert.hide();
    $scope.login_alert.rollmsg = function() {
        $scope.login_alert.fadeIn('slow', function() { 
            setTimeout(function() {
               $scope.login_alert.fadeOut('slow');
            }, 3000);
        });
    };
    $scope.login = function() {
        var creds = { "Username": $scope.username, "Password": $scope.password };
        $http.post('/login', creds).
            success(function(data, status) {
                $scope.extra_class = 'has-success';
                $scope.login_status = 'alert-success';
                $scope.login_message = data.Detail;
                $scope.next_url = '/#' + $location.path();
                console.log('Next URL = %s', $scope.next_url);
                $scope.login_alert.rollmsg();
                $(location).attr('href', $scope.next_url);
            }).
            error(function(data, status) {
                $scope.extra_class = 'has-error';
                $scope.login_status = 'alert-danger';
                $scope.login_message = data.Detail;
                $scope.login_alert.rollmsg();
            });
    };
}

function WizardControl($scope, $cookies, $location, wizardService, resourcesService) {
    console.log('Initialized control for %s', $location.path());
    $scope.params = {}; // No path params for wizard pages
    $scope.pools = {}; // We start with no pools
    $scope.context = wizardService.get_context();
    $scope.nextClicked = false;

    // Ensure our scope has a list of pools
    refreshPools($scope, resourcesService, true);

    $scope.add_host = function() {
        console.log('User added %s', $scope.context.newHost);
        $scope.context.hosts.push($scope.context.newHost);
        $scope.context.newHost = null;
    };
    $scope.next = function(wizardForm) {
        $scope.nextClicked = true;
        if (wizardForm == null || wizardForm.$valid) {
            console.log('Next called from %s', $location.path());
            var nextPath = wizardService.next_page($location.path());
            $location.path(nextPath);
        } else {
            console.log('Validation failed');
        }
    };
    $scope.cancel = function() {
        console.log('Cancel called from %s', $location.path());
        var nextPath = wizardService.cancel_page($location.path());
        $location.path(nextPath);
    };
    if ($cookies['ZCPToken'] === undefined) {
        $(location).attr('href', '/login');
    }
    $scope.form_class = 'form-group has-error';

    wizardService.fix_location($location);
}


// Controller for configuration
function ConfigurationControl($scope, $routeParams, servicesService) {
    $scope.name = "configuration";
    $scope.params = $routeParams;
    $scope.services = buildTable('Name', [
        { id: 'Id', name: 'Id'}, 
        { id: 'Name', name: 'Name'},
        { id: 'Description', name: 'Description'},
        { id: 'Startup', name: 'Startup'},
        { id: 'Instances', name: 'Instances'},
        { id: 'ImageId', name: 'Image Id'},
        { id: 'PoolId', name: 'Pool Id'},
        { id: 'DesiredState', name: 'Desired State'}
    ]);
    refreshServices($scope, servicesService, false);
}

function ActionControl($scope, $routeParams, resourcesService) {
    $scope.name = 'actions';
    $scope.params = $routeParams;
    $scope.newHost = {
        PoolId: $scope.params.poolId
    };
    $scope.newPool = {
        ParentId: $scope.params.poolId
    };

    $scope.add_host = function() {
        console.log('Adding host %s as child of pool %s', 
                    $scope.newHost.Name, $scope.newHost.PoolId);

        resourcesService.add_host($scope.newHost, function(data) {
            refreshHosts($scope, resourcesService);
        });
        // Reset for another add
        $scope.newHost = {
            PoolId: $scope.params.poolId
        };
    };

    $scope.add_pool = function() {
        console.log('Adding pool %s as child of pool %s', $scope.newPool.Id, $scope.params.poolId);
        resourcesService.add_pool($scope.newPool, function(data) {
            refreshPools($scope, resourcesService, false);
        });
        // Reset for another add
        $scope.newPool = {
            ParentId: $scope.params.poolId
        };
    };

    $scope.remove_pool = function() {
        console.log('Removing pool %s', $scope.params.poolId);
        resourcesService.remove_pool($scope.params.poolId, function(data) {
            var redirect = '#/resources';
            $('#removePool').on('hidden.bs.modal', function() {
                console.log('Redirecting to %s', redirect);
                $(location).attr('href', redirect);
            });

        });
    };

    $scope.remove_host = function() {
        console.log('Removing host %s', $scope.params.hostId);
        resourcesService.remove_host($scope.params.hostId, function(data) {
            var redirect = '#/pools/' + $scope.params.poolId;
            $('#removeHost').on('hidden.bs.modal', function() {
                console.log('Redirecting to %s', redirect);
                $(location).attr('href', redirect);
            });
        });
    };

    $scope.edit_pool = function() {
        console.log('Updating pool %s', $scope.params.poolId);
        resourcesService.update_pool($scope.params.poolId, $scope.pools.current, function(data) {
            refreshPools($scope, resourcesService, false);
        });
    };

    $scope.edit_host = function() {
        console.log('Updating host %s', $scope.params.hostId);
        resourcesService.update_host($scope.params.hostId, $scope.hosts.current, function(data) {
            refreshHosts($scope, resourcesService, false, false);
        });
    };
}

// Controller for resources
function ResourcesControl($scope, $routeParams, resourcesService) {
    $scope.name = "resources";
    $scope.params = $routeParams;

    $scope.pools = buildTable('Id', [
        { id: 'Id', name: 'Id'}, 
        { id: 'ParentId', name: 'Parent Id'},
        { id: 'CoreLimit', name: 'Core Limit'},
        { id: 'MemoryLimit', name: 'Memory Limit'},
        { id: 'Priority', name: 'Priority'}
    ]);
    $scope.set_order = set_order;
    $scope.get_order_class = get_order_class;
    $scope.click_pool = function(poolId) {
        var redirect = '#/pools/' + poolId;
        console.log('Redirecting to %s', redirect);
        $(location).attr('href', redirect);
    }
    $scope.hosts = {};

    refreshPools($scope, resourcesService, false);
    refreshHosts($scope, resourcesService, false, false);
}

// Controller for resources -> pool details
function PoolControl($scope, $routeParams, $http, resourcesService) {
    $scope.name = "pool-details";
    $scope.params = $routeParams;

    $scope.pools = {};
    refreshPools($scope, resourcesService, true);
    $scope.click_host = function(host) {
        var redirect = '#/pools/' + $scope.params.poolId + "/hosts/" + host;
        console.log('Redirecting to %s', redirect);
        $(location).attr('href', redirect);
    };
    $scope.set_order = set_order;
    $scope.get_order_class = get_order_class;
    $scope.hosts = buildTable('Name', [
        { id: 'Id', name: 'Id'}, 
        { id: 'Name', name: 'Name'},
        { id: 'Cores', name: 'Cores'},
        { id: 'Memory', name: 'Memory'},
        { id: 'IpAddr', name: 'IP Address'},
        { id: 'PrivateNetwork', name: 'Private Network'}
    ]);
    refreshHosts($scope, resourcesService, true, false);
}

// Controller for resources -> pool details -> host details
function HostControl($scope, $routeParams, $http, resourcesService) {
    $scope.name = "host-details"
    $scope.params = $routeParams;
    $scope.pools = {};
    refreshPools($scope, resourcesService, true);
    $scope.hosts = {};
    console.log('In scope for host ' + $scope.params.hostId);
    refreshHosts($scope, resourcesService, true, true);
}

// Controller for top nav
function NavbarControl($scope, $http, $cookies) {
    if ($cookies['ZCPToken'] === undefined) {
        $(location).attr('href', '/login');
    }
    $scope.management = 'Management';
    $scope.configuration = 'Configuration';
    $scope.resources = 'Resources';
    $scope.username = $cookies['ZUsername'];
    $scope.brand = { url: '#/entry', label: 'Control Plane' };
    $scope.navlinks = [
        { url: '#/services', label: 'Configuration' },
        { url: '#/resources', label: 'Resources' }
    ];
    $scope.logout = function() {
        $http.delete('/login').
            success(function(data, status) {
                var redirect = next_url(data);
                console.log('Next URL = %s', redirect);
                $(location).attr('href', redirect);
            }).
            error(function(data, status) {
                console.log('Unable to log out. Were you logged in to begin with?');
            });
    };
}

/*******************************************************************************
 * Helper functions
 ******************************************************************************/

function WizardService() {
    var wizard_data;
    var _get_wizard_data = function() {
        if (wizard_data === undefined) {
            wizard_data = {
                installType: 'local',
                localInstallType: 'Resource Manager',
                installOptions: [
                    'Resource Manager',
                    'Impact',
                    'Analytics'
                ],
                destination: 'default',
                flow: [
                    '/wizard/start', 
                    '/wizard/page1', 
                    '/wizard/page2', 
                    '/wizard/finish'
                ],
                cancel: '/',
                done: {}
            };
        }
        return wizard_data;
    };
    var _fix_location = function($location) {
        
        var d = _get_wizard_data();
        var pageIndex = 0;
        for (var i=0; i < d.flow.length; i++) {
            if (d.flow[i] === $location.path()) {
                // Found current element.
                pageIndex = i;
                break;
            }
        }
        
        var needToRedirect = false;
        while (d.done[d.flow[pageIndex -1]] !== true && pageIndex > 0) {
            pageIndex -= 1;
            needToRedirect = true;
        }

        if (needToRedirect) {
            var redirect = d.flow[pageIndex];
            console.log('Requirements not met so redirecting to: %s', redirect);
            $location.path(redirect);
        } else {
            console.log('Requirements met for page %s', $location.path());
        }
    };

    var _current_done = function(currentPath) {
        var d = _get_wizard_data();
        d.done[currentPath] = true;
    };

    var _cancel_page = function(currentPath) {
        var d = _get_wizard_data();
        return d.cancel;
    };

    var _next_page = function(currentPath) {
        _current_done(currentPath);
        var d = _get_wizard_data();
        var pageIndex = 0;
        for (var i=0; i < d.flow.length; i++) {
            if (d.flow[i] === currentPath) {
                // Found current element.
                pageIndex = i;
                break;
            }
        }
        return d.flow[pageIndex + 1];
    };

    return {
        get_context: _get_wizard_data,
        fix_location: _fix_location,
        next_page: _next_page,
        cancel_page: _cancel_page
    };
}

function ServicesService($http) {
    var cached_services;
    var _get_services = function(callback) {
        $http.get('/services').
            success(function(data, status) {
                console.log('Retrieved list of services');
                cached_services = data;
                callback(data);
            }).
            error(function(data, status) {
                console.log('Unable to retrieve services');
            });
    };

    return {
        get_services: function(cacheOk, callback) {
            if (cacheOk && cached_services) {
                console.log('Using cached services');
                callback(cached_services);
            } else {
                _get_services(callback);
            }
        }
    }
}

function ResourcesService($http) {
    var cached_pools;
    var cached_hosts_for_pool = {};
    var cached_hosts;

    // Real implementation for acquiring list of resource pools
    var _get_pools = function(callback) {
        $http.get('/pools').
            success(function(data, status) {
                console.log('Retrieved list of pools');
                cached_pools = data
                callback(data);
            }).
            error(function(data, status) {
                console.log('Unable to retrieve list of pools');
            });
    };
    var _get_hosts_for_pool = function(poolId, callback) {
        $http.get('/pools/' + poolId + '/hosts').
            success(function(data, status) {
                console.log('Retrieved hosts for pool %s', poolId);
                cached_hosts_for_pool[poolId] = data;
                callback(data);
            }).
            error(function(data, status) {
                console.log('Unable to retrieve hosts for pool %s', poolId);
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
                console.log('Unable to retrieve host details');
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
                    console.log('Adding pool failed: ' + JSON.stringify(data));
                });
        },

        /*
         * Updates existing resource pool.
         *
         * @param {string} poolId Unique identifier for pool to be edited.
         * @param {object} editedPool New pool details for provided poolId.
         * @param {function} callback Update result passed to callback on success.
         */
        update_pool: function(poolId, editedPool, callback) {
            $http.post('/pools/' + poolId, editedPool).
                success(function(data, status) {
                    console.log('Updated pool ' + poolId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Updating pool failed: ' + JSON.stringify(data));
                });
        },

        /*
         * Deletes existing resource pool.
         *
         * @param {string} poolId Unique identifier for pool to be removed.
         * @param {function} callback Delete result passed to callback on success.
         */
        remove_pool: function(poolId, callback) {
            $http.delete('/pools/' + poolId).
                success(function(data, status) {
                    console.log('Removed pool ' + poolId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Removing pool failed: ' + JSON.stringify(data));
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
                    console.log('Adding host failed: ' + JSON.stringify(data));
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
            $http.post('/hosts/' + hostId, editedHost).
                success(function(data, status) {
                    console.log('Updated host ' + hostId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Updating host failed: ' + JSON.stringify(data));
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
                    console.log('Removed host' + hostId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Removing host failed: ' + JSON.stringify(data));
                });
        },

        /*
         * Get the list of hosts belonging to a specified pool.
         * 
         * @param {boolean} cacheOk Whether or not cached data is OK to use.
         * @param {string} poolId Unique identifier for pool to use.
         * @param {function} callback List of hosts pass to callback on success.
         */
        get_hosts_for_pool: function(cacheOk, poolId, callback) {
            if (cacheOk && cached_hosts_for_pool[poolId]) {
                callback(cached_hosts_for_pool[poolId]);
            } else {
                _get_hosts_for_pool(poolId, callback);
            }
        }
    };
}

function refreshServices($scope, servicesService, cacheOk) {
    servicesService.get_services(cacheOk, function(allServices) {
        $scope.services.data = allServices;
        $scope.services.mapped = {};
        // Create a Map(Id -> Service)
        allServices.map(function(elem) {
            $scope.services.mapped[elem.Id] = elem;
        });
    });
}

function refreshPools($scope, resourcesService, cachePools) {
    resourcesService.get_pools(cachePools, function(allPools) {
        $scope.pools.data = allPools;//map_to_array(allPools);
        if ($scope.params.poolId !== undefined) {
            $scope.pools.current = allPools[$scope.params.poolId];
            console.log('Current pool: %s', JSON.stringify($scope.pools.current));
        }
    });
}

function refreshHosts($scope, resourcesService, cacheHosts, cacheHostsPool) {
    console.log('Reacquiring list of hosts');
    resourcesService.get_hosts(cacheHosts, function(allHosts) {
        // This is a Map(Id -> Host)
        $scope.hosts.rawdata = allHosts;
        // Build array of Hosts relevant to the current pool
        $scope.hosts.data = [];
        console.log('Refresh for pool ' + $scope.params.poolId + 
                    ' and host ' + $scope.params.hostId);

        if ($scope.params.poolId !== undefined) {
            resourcesService.get_hosts_for_pool(cacheHostsPool, $scope.params.poolId, function(hostsForPool) {
                // hostsForPool is Array(PoolHost)
                for (var i=0; i < hostsForPool.length; i++) {
                    var currentHost = allHosts[hostsForPool[i].HostId];
                    $scope.hosts.data.push(currentHost);
                    if ($scope.params.hostId === currentHost.Id) {
                        $scope.hosts.current = currentHost;
                        console.log('Current host: %s', JSON.stringify($scope.hosts.current));
                    }
                }
            });
        }
    });
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

/*
 * Helper function acquires next URL from data that looks like this:
 *
   {
     ...,
     'Links': [ { 'Next': '/some/url' }, ... ]
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
    return 'glyphicon pull-right ' + table.sort_icons[order] + 
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
        get_order_class: get_order_class
    };
}

