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
            when('/login', {
                templateUrl: '/static/partials/login.html',
                controller: LoginControl}).
            when('/services', {
                templateUrl: '/static/partials/services.html',
                controller: ConfigurationControl}).
            when('/services/:serviceId', {
                templateUrl: '/static/partials/service-details.html',
                controller: ServiceControl}).
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
            when('/hosts', {
                templateUrl: '/static/partials/view-hosts.html',
                controller: ResourcesControl}).
            otherwise({redirectTo: '/entry'});
    }]).
    factory('resourcesService', ResourcesService).
    factory('wizardService', WizardService).
    factory('servicesService', ServicesService).
    factory('authService', AuthService).
    directive('zDef', function ($compile) {
        // This directive builds a definition list for the object named by 
        // 'to-define' using the fields enumerated in 'define-headers'
        return {
            // This directive appears as an attribute
            restrict: 'A', 
            compile: function(tElem, tAttr) {
                return function($scope, $elem, $attrs) {
                    // called whenever the object named in 'define-headers' changes
                    $scope.$watch($attrs.defineHeaders, function(newVal, oldVal) {
                        var defineHeaders = newVal;
                        if (defineHeaders) {
                            var sb = '';
                            for(var i=0; i < defineHeaders.length; i++) {
                                sb += '<dt>' + defineHeaders[i].name +'</dt>';
                                sb += '<dd>{{' + $attrs.toDefine + '.' + defineHeaders[i].id + '}}</dd>';
                            }
                            tElem.html(sb);
                            $compile(tElem.contents())($scope);
                        }
                    });
                };
            }
        };
    });


function EntryControl($scope, authService) {
    console.log('Loading entry');
    authService.checkLogin($scope);
    $scope.mainlinks = [
        { url: '#/wizard/start', label: 'Install' },
        { url: '#/services', label: 'Configure services' },
        { url: '#/resources', label: 'Manage resources' }
    ];
}

// Used by /login view
function LoginControl($scope, $http, $location, authService) {
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
    // Makes XHR POST with contents of login form
    $scope.login = function() {
        var creds = { "Username": $scope.username, "Password": $scope.password };
        $http.post('/login', creds).
            success(function(data, status) {
                // Ensure that the auth service knows that we are logged in
                authService.login(true);
                // Redirect to main page
                $location.path('/entry');
            }).
            error(function(data, status) {
                console.log('Login failed');
                // Ensure that the auth service knows that the login failed
                authService.login(false);
                $scope.extra_class = 'has-error';
                $scope.login_status = 'alert-danger';
                $scope.login_message = data.Detail;
                $scope.login_alert.rollmsg();
            });
    };
}

function WizardControl($scope, $location, wizardService, resourcesService, authService) {
    // Ensure that if the user is not logged in, we show the /login page
    authService.checkLogin($scope);

    console.log('Initialized control for %s', $location.path());
    $scope.params = {}; // No path params for wizard pages
    $scope.pools = {}; // We start with no pools
    $scope.context = wizardService.get_context();
    $scope.nextClicked = false;

    // Ensure our scope has a list of pools
    refreshPools($scope, resourcesService, true);
    
    // The next function checks form validity then gets location from wizardService
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
    // All wizard pages have some kind of cancel function. Delegate location.
    $scope.cancel = function() {
        console.log('Cancel called from %s', $location.path());
        var nextPath = wizardService.cancel_page($location.path());
        $location.path(nextPath);
    };

    // This call ensures that the previous page was processed.
    wizardService.fix_location($location);
}


// Controller for configuration
function ConfigurationControl($scope, $routeParams, $location, servicesService, authService) {
    // Ensure that if the user is not logged in, we show the /login page
    authService.checkLogin($scope);

    $scope.name = "configuration";
    $scope.params = $routeParams;
    $scope.breadcrumbs = [
        { label: 'Configuration', itemClass: 'active' }
    ];

    // Build metadata necessary to display a table of services
    $scope.services = buildTable('Name', [
        { id: 'Name', name: 'Name'},
        { id: 'Description', name: 'Description'},
        { id: 'PoolId', name: 'Pool Id'},
        { id: 'Instances', name: 'Instances'}
    ]);

    // Create a function for when the user clicks on a service
    $scope.click_service = function(serviceId) {
        var redirect = '/services/' + serviceId;
        $location.path(redirect);
    };

    // Get the actual data about the services available
    refreshServices($scope, servicesService, false);
}

// Controller for configuration
function ServiceControl($scope, $routeParams, servicesService, authService) {
    // Ensure that if the user is not logged in, we show the /login page
    authService.checkLogin($scope);

    $scope.name = "configuration";
    $scope.params = $routeParams;
    $scope.breadcrumbs = [
        { label: 'Configuration', url: '#/services', itemClass: '' },
        { label: $scope.params.serviceId, itemClass: 'active' }
    ];

    // Create metadata necessary to display service details
    $scope.services = {
        headers: [
            { name: 'Id', id: 'Id' },
            { name: 'Name', id: 'Name' },
            { name: 'Description', id: 'Description' },
            { name: 'Pool Id', id: 'PoolId' },
            { name: 'Startup Command', id: 'Startup' },
            { name: 'Instances', id: 'Instances' },
            { name: 'Desired State', id: 'DesiredState' }
        ]
    };

    // Ensure that we have service data populated
    refreshServices($scope, servicesService, true);
}

// Common controller for resource action buttons
function ActionControl($scope, $routeParams, $location, resourcesService, servicesService) {
    $scope.name = 'actions';
    $scope.params = $routeParams;

    // New hosts should belong to the current pool by default
    $scope.newHost = {
        PoolId: $scope.params.poolId
    };

    // New pools should belong to the current pool by default
    $scope.newPool = {
        ParentId: $scope.params.poolId
    };

    // Just create a stub for new services
    $scope.newService = {};

    // Function for adding new hosts
    $scope.add_host = function() {
        console.log('Adding host %s as child of pool %s', 
                    $scope.newHost.Name, $scope.newHost.PoolId);

        resourcesService.add_host($scope.newHost, function(data) {
            // After adding, refresh our list
            refreshHosts($scope, resourcesService, false);
        });
        // Reset for another add
        $scope.newHost = {
            PoolId: $scope.params.poolId
        };
    };

    // Function for adding new pools
    $scope.add_pool = function() {
        console.log('Adding pool %s as child of pool %s', $scope.newPool.Id, $scope.params.poolId);
        resourcesService.add_pool($scope.newPool, function(data) {
            // After adding, refresh our list
            refreshPools($scope, resourcesService, false);
        });
        // Reset for another add
        $scope.newPool = {
            ParentId: $scope.params.poolId
        };
    };

    // Function for removing the current pool
    $scope.remove_pool = function() {
        console.log('Removing pool %s', $scope.params.poolId);
        resourcesService.remove_pool($scope.params.poolId, function(data) {

            // The UI can be weird if we don't wait for the modal to hide
            // before we change the path
            $('#removePool').on('hidden.bs.modal', function() {
                var redirect = '/resources';
                console.log('Redirecting to %s', redirect);
                $location.path(redirect);

                // Because this is happening in a weird place, the $scope 
                // seems to need a hint
                $scope.$apply();
            });

        });
    };

    // Function for removing the current host
    $scope.remove_host = function() {
        console.log('Removing host %s', $scope.params.hostId);
        resourcesService.remove_host($scope.params.hostId, function(data) {

            // The UI can be weird if we don't wait for the modal to hide
            // before we change the path
            $('#removeHost').on('hidden.bs.modal', function() {
                var redirect = '/pools/' + $scope.params.poolId;
                console.log('Redirecting to %s', redirect);
                $location.path(redirect);

                // Because this is happening in a weird place, the $scope 
                // seems to need a hint
                $scope.$apply();
            });
        });
    };

    // Function for editing the current pool
    $scope.edit_pool = function() {
        console.log('Updating pool %s', $scope.params.poolId);
        resourcesService.update_pool($scope.params.poolId, $scope.editPool, function(data) {
            // After the edit succeeds, refresh the list
            refreshPools($scope, resourcesService, false);
        });
    };

    // Function for editing the current host
    $scope.edit_host = function() {
        console.log('Updating host %s', $scope.params.hostId);
        resourcesService.update_host($scope.params.hostId, $scope.editHost, function(data) {
            // After the edit succeeds, refresh the list
            refreshHosts($scope, resourcesService, false, false);
        });
    };

    // Function for adding a new host
    $scope.add_service = function() {
        console.log('Adding service %s', $scope.newService.Name);
        servicesService.add_service($scope.newService, function(data) {
            // After the add succeeds, refresh the list
            refreshServices($scope, servicesService, false);
        });
        // Reset for another add
        $scope.newService = {};
    };

    // Function for editing the current service
    $scope.edit_service = function() {
        console.log('Editing service %s', $scope.services.current.Name);
        servicesService.update_service($scope.params.serviceId, $scope.editService, function(data) {
            // After the edit succeeds, refresh the list
            refreshServices($scope, servicesService, false);
        });
    };

    // Function for removing the current service
    $scope.remove_service = function() {
        console.log('Removing service %s', $scope.params.serviceId);
        servicesService.remove_service($scope.params.serviceId, function(data) {

            // The UI can be weird if we don't wait for the modal to hide
            // before we change the path
            $('#removeService').on('hidden.bs.modal', function() {
                console.log('redirecting to /services');
                $location.path('/services');

                // Because this is happening in a weird place, the $scope 
                // seems to need a hint
                $scope.$apply();
            });
        });

    }
}

// Controller for resources
function ResourcesControl($scope, $routeParams, $location, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "resources";
    $scope.params = $routeParams;
    $scope.breadcrumbs = [
        { label: 'Resources', itemClass: 'active' }
    ];
    // Build metadata for displaying a list of pools
    $scope.pools = buildTable('Id', [
        { id: 'Id', name: 'Id'}, 
        { id: 'ParentId', name: 'Parent Id'},
        { id: 'Priority', name: 'Priority'}
    ]);

    // Create function for selecting a pool
    $scope.click_pool = function(poolId) {
        var redirect = '/pools/' + poolId;
        $location.path(redirect);
    }
    $scope.hosts = {};
    $scope.toggleCollapsed = function(poolId) {
        console.log('pool clicked %s', poolId);
        var toggled = $scope.pools.mapped[poolId];
        toggled.collapsed = !toggled.collapsed;
        if (toggled.children === undefined) {
            return;
        }
        if (toggled.collapsed) {
            toggled.icon = 'glyphicon glyphicon-plus link';
            toggled.childrenClass = 'hidden';
        } else {
            toggled.icon = 'glyphicon glyphicon-minus link';
            toggled.childrenClass = 'nav-tree';
        }

    };

    // Ensure we have a list of pools
    refreshPools($scope, resourcesService, false);
    // Also ensure we have a list of hosts
    refreshHosts($scope, resourcesService, false, false);
}

// Controller for resources -> pool details
function PoolControl($scope, $routeParams, $http, $location, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "pool-details";
    $scope.params = $routeParams;
    $scope.breadcrumbs = [
        { label: 'Resources', url: '#/resources' },
        { label: $scope.params.poolId, itemClass: 'active' }
    ];

    // Build metadata for displaying pool details
    $scope.pools = {
        headers: [
            { id: 'Id', name: 'Id'}, 
            { id: 'ParentId', name: 'Parent Id'},
            { id: 'CoreLimit', name: 'Core Limit'},
            { id: 'MemoryLimit', name: 'Memory Limit'},
            { id: 'Priority', name: 'Priority'}
        ]
    };
    // Populate list of pools
    refreshPools($scope, resourcesService, true);

    // Create function for selecting a host
    $scope.click_host = function(pool, host) {
        var redirect = '/pools/' + pool + "/hosts/" + host;
        console.log('Redirecting to %s', redirect);
        $location.path(redirect);
    };

    // Build metadata for displaying a list of hosts
    $scope.hosts = buildTable('Name', [
        { id: 'Name', name: 'Name'},
        { id: 'IpAddr', name: 'IP Address'},
        { id: 'PrivateNetwork', name: 'Private Network'}
    ]);

    // Populate list of hosts
    refreshHosts($scope, resourcesService, true, false);
}

// Controller for resources -> pool details -> host details
function HostControl($scope, $routeParams, $http, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "host-details"
    $scope.params = $routeParams;
    $scope.breadcrumbs = [
        { label: 'Resources', url: '#/resources' },
        { label: $scope.params.poolId, url: '#/pools/' + $scope.params.poolId },
        { label: $scope.params.hostId, itemClass: 'active' }
    ];

    $scope.pools = {};

    // Build metadata for displaying host details
    $scope.hosts = {
        headers: [
            { id: 'Id', name: 'Id'}, 
            { id: 'Name', name: 'Name'},
            { id: 'Cores', name: 'Cores'},
            { id: 'Memory', name: 'Memory'},
            { id: 'IpAddr', name: 'IP Address'},
            { id: 'PrivateNetwork', name: 'Private Network'}
        ]
    };

    // Populate list of pools
    refreshPools($scope, resourcesService, true);

    // Populate list of hosts
    refreshHosts($scope, resourcesService, true, true);
}


// Controller for top nav
function NavbarControl($scope, $http, $cookies, $location, authService) {
    $scope.management = 'Management';
    $scope.configuration = 'Configuration';
    $scope.resources = 'Resources';
    $scope.username = $cookies['ZUsername'];
    $scope.brand = { url: '#/entry', label: 'Control Plane' };
    $scope.navlinks = [
        { url: '#/services', label: 'Configuration' },
        { url: '#/resources', label: 'Resources' }
    ];

    // Create a logout function
    $scope.logout = function() {
        // Set internal state to logged out.
        authService.login(false);
        // Make http call to logout from server
        $http.delete('/login').
            success(function(data, status) {
                // On successful logout, redirect to /login
                $location.path('/login');
            }).
            error(function(data, status) {
                // On failure to logout, note the error
                console.log('Unable to log out. Were you logged in to begin with?');
            });
    };
}

/*******************************************************************************
 * Helper functions
 ******************************************************************************/

function WizardService() {
    // Wizard data is a context object that persists across controllers, but not
    // across full page loads. This is fine for what we want since we are 
    // controlling the view without full page loads.
    var wizard_data;

    // This is a lazy-loading accessor function that defines a default context
    // if one does not yet exist.
    var _get_wizard_data = function() {
        if (wizard_data === undefined) {
            wizard_data = {
                // Temporary: local or distributed
                installType: 'local',

                // Default install type
                localInstallType: 'Resource Manager',

                // List of products to install
                installOptions: [
                    'Resource Manager',
                    'Impact',
                    'Analytics'
                ],

                // Default pool
                destination: 'default',

                // The 'flow' field defines the order of the pages
                flow: [
                    '/wizard/start', 
                    '/wizard/page1', 
                    '/wizard/page2', 
                    '/wizard/finish'
                ],

                // Where to go when a user cancels the flow
                cancel: '/',

                // The 'done' field defines which pages have successfully been
                // completed.
                done: {}
            };
        }
        return wizard_data;
    };

    /*
     * The basic premise here is that we want to check to see if there is an 
     * entry for the previous page of the flow in our wizard_data.done object.
     * If yes then the current page is fine and we do nothing; otherwise repeat
     * repeat until we find a page with an appropriate wizard_data.done entry.
     */
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
        
        // Assume we don't need to redirect
        var needToRedirect = false;

        // We only redirect if this page is in the flow and the requirements are
        // not met.
        while (pageIndex > 0 && d.done[d.flow[pageIndex -1]] !== true) {
            // The requirements are not met for the current page. Set the 
            // current page to the previous page and check again.
            pageIndex -= 1;
            // We definitely need to redirect.
            needToRedirect = true;
        }

        if (needToRedirect) {
            var redirect = d.flow[pageIndex];
            console.log('Requirements not met so redirecting to: %s', redirect);
            $location.path(redirect);
        }
    };

    // This function is used to mark the current page as complete.
    var _current_done = function(currentPath) {
        var d = _get_wizard_data();
        d.done[currentPath] = true;
    };

    // This function is used to return the destination when a user cancels
    var _cancel_page = function(currentPath) {
        var d = _get_wizard_data();
        return d.cancel;
    };

    // Mark the current page as complete then return the path for the next page
    var _next_page = function(currentPath) {
        _current_done(currentPath);
        var d = _get_wizard_data();
        var pageIndex = 0;
        for (var i=0; i < d.flow.length; i++) {
            if (d.flow[i] === currentPath) {
                // Found current element.
                pageIndex = (i + 1);
                break;
            }
        }
        return d.flow[pageIndex];
    };

    // This function is a factory. Return an object.
    return {
        get_context: _get_wizard_data,
        fix_location: _fix_location,
        next_page: _next_page,
        cancel_page: _cancel_page
    };
}

function ServicesService($http, $location) {
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
                if (status === 401) {
                    unauthorized($location);
                }

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
        },

        add_service: function(service, callback) {
            console.log('Adding detail: %s', JSON.stringify(service));
            $http.post('/services/add', service).
                success(function(data, status) {
                    console.log('Added new service');
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Adding service failed: ' + JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        },

        update_service: function(serviceId, editedService, callback) {
            $http.post('/services/' + serviceId, editedService).
                success(function(data, status) {
                    console.log('Updated service ' + serviceId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Updating service failed: ' + JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        },

        remove_service: function(serviceId, callback) {
            $http.delete('/services/' + serviceId).
                success(function(data, status) {
                    console.log('Removed service ' + serviceId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Removing service failed: ' + JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        }

    }
}

function ResourcesService($http, $location) {
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
                if (status === 401) {
                    unauthorized($location);
                }
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
                console.log('Unable to retrieve host details');
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
                    if (status === 401) {
                        unauthorized($location);
                    }
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
                    if (status === 401) {
                        unauthorized($location);
                    }
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
                    console.log('Adding host failed: ' + JSON.stringify(data));
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
            $http.post('/hosts/' + hostId, editedHost).
                success(function(data, status) {
                    console.log('Updated host ' + hostId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Updating host failed: ' + JSON.stringify(data));
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
                    console.log('Removed host' + hostId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Removing host failed: ' + JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
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

function AuthService($cookies, $location) {
    var loggedIn = false;
    return {

        /*
         * Called when we have positively determined that a user is or is not
         * logged in.
         *
         * @param {boolean} truth Whether the user is logged in.
         */
        login: function(truth) {
            loggedIn = truth;
        },

        /*
         * Check whether the user appears to be logged in. Update path if not.
         *
         * @param {object} scope The 'loggedIn' property will be set if true
         */
        checkLogin: function($scope) {
            if (loggedIn) {
                $scope.loggedIn = true;
                return;
            }
            if ($cookies['ZCPToken'] !== undefined) {
                loggedIn = true;
                $scope.loggedIn = true;
            } else {
                unauthorized($location);
            }
        }
    };
}

function refreshServices($scope, servicesService, cacheOk) {
    // defend against empty scope
    if ($scope.services === undefined) {
        $scope.services = {};
    }
    servicesService.get_services(cacheOk, function(allServices) {
        $scope.services.data = allServices;
        $scope.services.mapped = {};
        // Create a Map(Id -> Service)
        allServices.map(function(elem) {
            $scope.services.mapped[elem.Id] = elem;
        });
        if ($scope.params && $scope.params.serviceId) {
            $scope.services.current = $scope.services.mapped[$scope.params.serviceId];
            $scope.editService = $.extend({}, $scope.services.current);
        }
    });
}

function refreshPools($scope, resourcesService, cachePools) {
    // defend against empty scope
    if ($scope.pools === undefined) {
        $scope.pools = {};
    }
    resourcesService.get_pools(cachePools, function(allPools) {
        $scope.pools.mapped = allPools;
        $scope.pools.data = map_to_array(allPools);
        $scope.pools.tree = [];

        for (var key in allPools) {
            var p = allPools[key];
            p.collapsed = false;
            p.childrenClass = "nav-tree";
            if (p.icon === undefined) {
                p.icon = 'glyphicon glyphicon-minus disabled';
            }
            var parent = allPools[p.ParentId];
            if (parent) {
                if (parent.children === undefined) {
                    parent.children = [];
                    parent.icon = 'glyphicon glyphicon-minus link';
                }
                console.log('Adding %s as child of %s', p.Id, p.ParentId);
                parent.children.push(p);
            } else {
                $scope.pools.tree.push(p);
            }
        }

        if ($scope.params && $scope.params.poolId) {
            $scope.pools.current = allPools[$scope.params.poolId];
            $scope.editPool = $.extend({}, $scope.pools.current);
        }
    });
}

function refreshHosts($scope, resourcesService, cacheHosts, cacheHostsPool) {
    // defend against empty scope
    if ($scope.hosts === undefined) {
        $scope.hosts = {};
    }
    resourcesService.get_hosts(cacheHosts, function(allHosts) {
        // This is a Map(Id -> Host)
        $scope.hosts.mapped = allHosts;
        // Build array of Hosts relevant to the current pool
        $scope.hosts.data = [];

        if ($scope.params && $scope.params.poolId) {
            resourcesService.get_hosts_for_pool(cacheHostsPool, $scope.params.poolId, function(hostsForPool) {
                // hostsForPool is Array(PoolHost)
                for (var i=0; i < hostsForPool.length; i++) {
                    var currentHost = allHosts[hostsForPool[i].HostId];
                    $scope.hosts.data.push(currentHost);
                    if ($scope.params.hostId === currentHost.Id) {
                        $scope.hosts.current = currentHost;
                        $scope.editHost = $.extend({}, $scope.hosts.current);
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

