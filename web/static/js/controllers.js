/*
 * This file provides the control aspect of the MVC paradigm for Control Plane.
 */
angular.module('controlplane', ['ngCookies']).
    config(['$routeProvider', function($routeProvider) {
        $routeProvider.
            when('/entry', {
                templateUrl: '/static/partials/main.html', 
                controller: EntryControl}).
            when('/management', {
                templateUrl: '/static/partials/management.html', 
                controller: ManagementControl}).
            when('/management/:typeId', {
                templateUrl: '/static/partials/management.html', 
                controller: ManagementControl}).
            when('/configuration', {
                templateUrl: '/static/partials/configuration.html',
                controller: ConfigurationControl}).
            when('/resources', {
                templateUrl: '/static/partials/resources.html',
                controller: ResourcesControl}).
            when('/pools/:poolId', {
                templateUrl: '/static/partials/pool-details.html',
                controller: PoolControl}).
            otherwise({redirectTo: '/entry'});
    }]).
    /*
     * Dependency injected utility for working with resource pools.
     */
    factory('cpdata', function($http) {
        var cached_pools;
        var cached_hosts_for_pool = {};
        var cached_hosts;

        /*
         * Real implementation for acquiring list of resource pools
         */ 
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
                    console.log('Retrieved hosts for pool %s: %s', poolId, JSON.stringify(data));
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
                if (cacheOk && cached_hosts_for_pool) {
                    callback(cached_hosts_for_pool);
                } else {
                    _get_hosts_for_pool(poolId, callback);
                }
            }
        };
    });



/*
 * Controller for main splash
 */
function EntryControl($scope, $http) {
    $scope.startServer = function() {
        $http.post('/start', {}).
            success(function(data, status) {
                if (status === 200) {
                    console.log('Started service: %s', JSON.stringify(data));
                } else {
                    console.log('Got unexpected status %d', status);
                }
            }).
            error(function(data, status) {
                console.log('Failed with message %s and  status %d', JSON.stringify(data), status);
                var redirect = next_url(data);
                console.log('Next URL = %s', redirect);
                $(location).attr('href', redirect);
            });
    };
}

/*
 * Controller for management
 */
function ManagementControl($scope, $routeParams) {
    $scope.name = "management";
    $scope.params = $routeParams;
}

/*
 * Controller for configuration
 */
function ConfigurationControl($scope, $routeParams) {
    $scope.name = "configuration";
    $scope.params = $routeParams;
}

/*
 * Controller for resource pool side nav
 */
function PoolNavControl($scope, $routeParams, cpdata) {
    $scope.name = "pool-nav";
    $scope.params = $routeParams;
    $scope.newHost = {};
    $scope.newPool = {
        ParentId: $scope.params.poolId
    };

    $scope.editPool = $scope.pools.current;

    $scope.add_host = function() {
        console.log('Adding host %s as child of pool %s', $scope.newHost.Name, $scope.params.poolId);
        cpdata.add_host({
            Id: $scope.newHost.Name,
            Name: $scope.newHost.Name,
            IpAddr: $scope.newHost.Name,
            PoolId: $scope.params.poolId,
            Cores: 1,
            Memory: 12345
        }, function(data) {
            refreshHosts($scope, cpdata);
        });
        // Reset for another add
        $scope.newHost = {};
    };

    $scope.add_pool = function() {
        console.log('Adding pool %s as child of pool %s', $scope.newPool.Id, $scope.params.poolId);
        cpdata.add_pool($scope.newPool, function(data) {
            refreshPools($scope, cpdata);
        });
        // Reset for another add
        $scope.newPool = {
            ParentId: $scope.params.poolId
        };
    };

    $scope.remove_pool = function() {
        console.log('Removing pool %s', $scope.params.poolId);
        cpdata.remove_pool($scope.params.poolId, function(data) {
            var redirect = next_url(data);
            console.log('Next URL = %s', redirect);
            $(location).attr('href', redirect);
        });
    };

    $scope.remove_host = function() {
        console.log('Removing host %s', $scope.params.hostId);
        cpdata.remove_host($scope.params.hostId, function(data) {
            refreshHosts($scope, cpdata);
        });
    };

    $scope.edit_pool = function() {
        console.log('Updating pool %s', $scope.params.poolId);
        cpdata.update_pool($scope.params.poolId, $scope.editPool, function(data) {
            refreshPools($scope, cpdata);
        });
    };

    $scope.edit_host = function() {
        console.log('TODO: Updating host %s', $scope.params.hostId);
    };
}

/*
 * Controller for resources
 */
function ResourcesControl($scope, $routeParams, cpdata) {
    $scope.name = "resources";
    $scope.params = $routeParams;

    $scope.pools = {};
    $scope.hosts = {};

    refreshPools($scope, cpdata, false);
    refreshHosts($scope, cpdata, false, false);
}

/*
 * Controller for resources -> pool details
 */
function PoolControl($scope, $routeParams, $http, cpdata) {
    $scope.name = "resources";
    $scope.params = $routeParams;

    $scope.pools = {};
    refreshPools($scope, cpdata, true);

    $scope.hosts = {
        sort: 'Name',
        sort_icons: {
            'Id': 'glyphicon-chevron-down',
            'Name': 'glyphicon-chevron-up',
            'Cores': 'glyphicon-chevron-down',
            'Memory': 'glyphicon-chevron-down',
            'IpAddr': 'glyphicon-chevron-down',
            'PrivateNetwork': 'glyphicon-chevron-down'
        },
        set_order: function(order) {
            // Reset the icon for the last order
            console.log('Resetting ' + $scope.resourceOrder + ' to down.');
            $scope.hosts.sort_icons[$scope.hosts.sort] = 'glyphicon-chevron-down';

            if ($scope.hosts.sort === order) {
                $scope.hosts.sort = "-" + order;
                $scope.hosts.sort_icons[$scope.hosts.sort] = 'glyphicon-chevron-down';
                console.log('Sorting by -' + order);
            } else {
                $scope.hosts.sort = order;
                $scope.hosts.sort_icons[$scope.hosts.sort] = 'glyphicon-chevron-up';
                console.log('Sorting by ' + order);
            }
        }
    };
    refreshHosts($scope, cpdata, true, false);
}

/*
 * Controller for top nav
 */
function NavbarControl($scope, $http, $cookies) {
    if ($cookies['ZCPToken'] === undefined) {
        $(location).attr('href', '/login');
    }
    $scope.brand = 'Control Plane';
    $scope.management = 'Management';
    $scope.configuration = 'Configuration';
    $scope.resources = 'Resources';
    $scope.username = $cookies['ZUsername'];
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

/*
 * Controller for login page
 */
function LoginControl($scope, $http) {
    $scope.brand_label = "SERVICE DYNAMICS";
    $scope.login_button_text = "Log In";
    $scope.next_url = $.url().param('next');
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
                if ($scope.next_url === undefined) {
                    $scope.next_url = next_url(data);
                }
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

function refreshPools($scope, cpdata, cachePools) {
    console.log('Reacquiring list of pools');
    cpdata.get_pools(cachePools, function(allPools) {
        $scope.pools.data = map_to_array(allPools);
        if ($scope.params.poolId !== undefined) {
            $scope.pools.current = allPools[$scope.params.poolId];
        }
    });
}

function refreshHosts($scope, cpdata, cacheHosts, cacheHostsPool) {
    console.log('Reacquiring list of hosts');
    cpdata.get_hosts(cacheHosts, function(allHosts) {
        // This is a Map(Id -> Host)
        $scope.hosts.rawdata = allHosts;
        // Build array of Hosts relevant to the current pool
        $scope.hosts.data = [];
        if ($scope.params.poolId !== undefined) {
            cpdata.get_hosts_for_pool(cacheHostsPool, $scope.params.poolId, function(hostsForPool) {
                // hostsForPool is Array(PoolHost)
                for (key in hostsForPool) {
                    $scope.hosts.data.push(allHosts[hostsForPool[key].HostId]);
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
