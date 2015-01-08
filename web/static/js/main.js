// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*******************************************************************************
 * Main module & controllers
 ******************************************************************************/
var controlplane = angular.module('controlplane', ['ngRoute', 'ngCookies','ngDragDrop','pascalprecht.translate', 'angularMoment', 'zenNotify', 'serviceHealth', 'ui.datetimepicker', 'modalService', 'angular-data.DSCacheFactory', 'stealthInput', 'ui.codemirror', 'sticky', 'graphPanel', 'servicesFactory']);

controlplane.
    config(['$routeProvider', function($routeProvider) {
        $routeProvider.
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
            otherwise({redirectTo: '/apps'});
    }]).
    config(['$translateProvider', function($translateProvider) {

        $translateProvider.useStaticFilesLoader({
            prefix: '/static/i18n/',
            suffix: '.json'
        });
        $translateProvider.preferredLanguage('en_US');
    }]).
    config(['DSCacheFactoryProvider', function(DSCacheFactory){
        DSCacheFactory.setCacheDefaults({
            // Items will not be deleted until they are requested
            // and have expired
            deleteOnExpire: 'passive',

            // This cache will clear itself every hour
            cacheFlushInterval: 3600000,

            // This cache will sync itself with localStorage
            storageMode: 'memory'
         });
    }]).
    /**
     * Default Get requests to no caching
     **/
    config(["$httpProvider", function($httpProvider){
        //initialize get if not there
        if (!$httpProvider.defaults.headers.get) {
            $httpProvider.defaults.headers.get = {};
        }
        $httpProvider.defaults.headers.get['Cache-Control'] = 'no-cache';
        $httpProvider.defaults.headers.get['Pragma'] = 'no-cache';
        $httpProvider.defaults.headers.get['If-Modified-Since'] = 'Mon, 26 Jul 1997 05:00:00 GMT';
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
    }).
    factory('authService', AuthService).
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
                return validData[elem[field]] !== null;
            });
        };
    }).
    filter('toGB', function(){
        return function(input){
            return (input/1073741824).toFixed(2) + " GB";
        };
    }).
    filter('cut', function(){
        return function (value, wordwise, max, tail) {
            if (!value) return '';

            max = parseInt(max, 10);
            if (!max) return value;
            if (value.length <= max) return value;

            value = value.substr(0, max);
            if (wordwise) {
                            var lastspace = value.lastIndexOf(' ');
                            if (lastspace != -1) {
                                                value = value.substr(0, lastspace);
                                            }
                        }

            return value + (tail || ' …');
        };
    }).
    filter('prettyDate', function(){
        return function(dateString){
            return moment(dateString).format('MMM Do YYYY, hh:mm:ss');
        };
    });


// set verbosity of console.logs
var DEBUG = false;

function AuthService($cookies, $cookieStore, $location, $http, $notification) {
    var loggedIn = false;
    var userName = null;

    var setLoggedIn = function(truth, username) {
        loggedIn = truth;
        userName = username;
    };
    return {

        /*
         * Called when we have positively determined that a user is or is not
         * logged in.
         *
         * @param {boolean} truth Whether the user is logged in.
         */
        setLoggedIn: setLoggedIn,

        login: function(creds, successCallback, failCallback){
            $http.post('/login', creds).
                success(function(data, status) {
                    // Ensure that the auth service knows that we are logged in
                    setLoggedIn(true, creds.Username);

                    successCallback();
                }).
                error(function(data, status) {
                    // Ensure that the auth service knows that the login failed
                    setLoggedIn(false);

                    failCallback();
                });
        },

        logout: function(){
            $http.delete('/login').
                success(function(data, status) {
                    // On successful logout, redirect to /login
                    $location.path('/login');
                }).
                error(function(data, status) {
                    // On failure to logout, note the error
                    // TODO error screen
                    console.error('Unable to log out. Were you logged in to begin with?');
                });
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


/*
 * manage pools
 * TODO - move pools to separate service
 */
var POOL_ICON_CLOSED = 'glyphicon glyphicon-play btn-link';
var POOL_ICON_OPEN = 'glyphicon glyphicon-play rotate-down btn-link';
var POOL_CHILDREN_CLOSED = 'hidden';
var POOL_CHILDREN_OPEN = 'nav-tree';
function refreshPools($scope, resourcesFactory, cachePools, extraCallback) {
    // defend against empty scope
    if ($scope.pools === undefined) {
        $scope.pools = {};
    }
    if(DEBUG) console.log('Refreshing pools');
    resourcesFactory.get_pools(cachePools, function(allPools) {
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

function getFullPath(allPools, pool) {
    if (!allPools || !pool.ParentId || !allPools[pool.ParentId]) {
        return pool.ID;
    }
    return getFullPath(allPools, allPools[pool.ParentId]) + " > " + pool.ID;
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
        retVal = retVal.concat(flattenTree(depth + 1, current.children[i], sortFunction));
    }
    return retVal;
}


/*
 * Manage hosts
 * TODO - move host management into a separate service
 */
function refreshHosts($scope, resourcesFactory, cacheHosts, extraCallback) {
    // defend against empty scope
    if ($scope.hosts === undefined) {
        $scope.hosts = {};
    }

    resourcesFactory.get_hosts(cacheHosts, function(allHosts) {
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
function refreshRunningForHost($scope, resourcesFactory, hostId) {
    if ($scope.running === undefined) {
        $scope.running = {};
    }

    resourcesFactory.get_running_services_for_host(hostId, function(runningServices) {
        $scope.running.data = runningServices;
        for (var i=0; i < runningServices.length; i++) {
            runningServices[i].DesiredState = 1; // All should be running
            runningServices[i].Deployment = 'successful'; // TODO: Replace
        }
    });
}
// add pool path to host
function fix_pool_paths($scope) {
    if ($scope.pools && $scope.pools.mapped && $scope.hosts && $scope.hosts.all) {
        for(var i=0; i < $scope.hosts.all.length; i++) {
            var host = $scope.hosts.all[i];
            host.fullPath = $scope.pools.mapped[host.PoolID].fullPath;
        }
    }
}


/*
 * Functions for setting up grid views
 * TODO - create angular controller for grids
 */
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
    };
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


/*
 * Helper and utility functions
 */
function map_to_array(data) {
    var arr = [];
    for (var key in data) {
        arr[arr.length] = data[key];
    }
    return arr;
}

function unauthorized($location) {
    console.error('You don\'t appear to be logged in.');
    // show the login page and then refresh so we lose any incorrect state. CC-279
    window.location.href = "/#/login";
    window.location.reload();
}

function indentClass(depth) {
    return 'indent' + (depth -1);
}

function downloadFile(url){
    window.location = url;
}

function getModeFromFilename(filename){
    var re = /(?:\.([^.]+))?$/;
    var ext = re.exec(filename)[1];
    var mode;
    switch(ext) {
        case "conf":
            mode="properties";
            break;
        case "xml":
            mode = "xml";
            break;
        case "yaml":
            mode = "yaml";
            break;
        case "txt":
            mode = "plain";
            break;
            case "json":
            mode = "javascript";
            break;
        default:
            mode = "shell";
            break;
    }

    return mode;
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
    $translate.use(ln);
}
