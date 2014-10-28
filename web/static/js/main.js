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
var controlplane = angular.module('controlplane', ['ngRoute', 'ngCookies','ngDragDrop','pascalprecht.translate', 'angularMoment', 'zenNotify', 'serviceHealth', 'modalService', 'angular-data.DSCacheFactory']);

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
            when('/isvcs', {
                templateUrl: '/static/partials/view-isvcs.html',
                controller: IsvcsControl
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
    }
);

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
    $translate.use(ln);
}

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
  vhosts.sort(function(a, b){ return (a.Name < b.Name ? -1 : 1); });
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
          });
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
    servicesService.update_services(function(topServices, mappedServices) {
        $scope.services.data = topServices;
        $scope.services.mapped = mappedServices;

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

function toggleRunning(app, status, servicesService, skipChildren) {
    var serviceId;

    // if app is an instance, use ServiceId
    if(isInstanceOfService(app)){
        serviceId = app.ServiceID;

    // else, app is a service, so use ID
    } else {
        serviceId = app.ID;
    }

    var newState = -1;
    switch(status) {
        case 'start': newState = 1; break;
        case 'stop': newState = 0; break;
        case 'restart': newState = -1; break;
    }

    app.DesiredState = newState;

    // stop service
    if ((newState === 0) || (newState === -1)) {
        servicesService.stop_service(serviceId, function(){}, skipChildren);
    }

    // start service
    if ((newState === 1) || (newState === -1)) {
        servicesService.start_service(serviceId, function(){}, skipChildren);
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
        $scope.running = {data:[]};
    }

    resourcesService.get_running_services_for_service(serviceId, function(runningServices) {
        // merge running.data with runningServices without creating a new object

        // create a list of current running.data ids to check for removals
        var oldIds = $scope.running.data.map(function(el){return el.ID;}),
            running = $scope.running.data;

        runningServices.forEach(function(runningService){
            var oldServiceIndex = oldIds.indexOf(runningService.ID),
                oldService;

            // if this guy is already in the running list
            if(oldServiceIndex !== -1){
                oldService = running[oldServiceIndex];
                
                // merge changes in
                for(var i in runningService){
                    oldService[i] = runningService[i];
                }

                // remove this id from the oldIds list
                oldIds[oldServiceIndex] = null;
                
            // else this is a new running service, so add it
            } else {
                running.push(runningService);
            }
        });

        // any ids left in oldIds should be removed from running
        for(var i = running.length - 1; i >= 0; i--){
            if(~oldIds.indexOf(running[i].ID)){
                console.log("removing old id", running[i].ID, "at index", i);
                running.splice(i, 1);
            }
        }

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

function unauthorized($location) {
    console.error('You don\'t appear to be logged in.');
    // show the login page and then refresh so we lose any incorrect state. CC-279
    window.location.href = "/#/login";
    window.location.reload();
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

// determines if an object is an instance of a service
function isInstanceOfService(service){
    return "InstanceID" in service;
}

// keep notifications stuck to bottom of nav, or top of window
// if nav is out ovf view.
var $window = $(window);
$window.on("scroll", function(){
    var currScrollTop = $window.scrollTop(),
        $notifications = $("#notifications");

    if(currScrollTop > 0){
        var top = Math.max(80 - currScrollTop, 0);
        $notifications.css("top", top+"px");
    }else{
        $notifications.css("top", "80px");
    }
});
