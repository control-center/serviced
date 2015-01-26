/* globals DEBUG: true */

/* miscUtils.js
 * miscellaneous utils and stuff that
 * doesn't quite fit in elsewhere
 */
(function(){
    "use strict";

    angular.module("miscUtils", [])
    .factory("miscUtils", [
    function(){

        var utils = {
            /*
             * manage pools
             * TODO - move pools to separate service
             */
             refreshPools: function($scope, resourcesFactory, cachePools, extraCallback) {
                var POOL_ICON_OPEN = 'glyphicon glyphicon-play rotate-down btn-link';

                // defend against empty scope
                if ($scope.pools === undefined) {
                    $scope.pools = {};
                }
                if(DEBUG){
                    console.log('Refreshing pools');
                }
                resourcesFactory.get_pools(cachePools, function(allPools) {
                    $scope.pools.mapped = allPools;
                    $scope.pools.data = utils.map_to_array(allPools);
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
                            p.fullPath = utils.getFullPath(allPools, p);
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

                    $scope.pools.flattened = utils.flattenTree(0, flatroot);

                    if (extraCallback) {
                        extraCallback();
                    }
                });
            },
            getFullPath: function(allPools, pool) {
                if (!allPools || !pool.ParentId || !allPools[pool.ParentId]) {
                    return pool.ID;
                }
                return utils.getFullPath(allPools, allPools[pool.ParentId]) + " > " + pool.ID;
            },

            /*
             * Starting at some root node, recurse through children,
             * building a flattened array where each node has a depth
             * tracking field 'zendepth'.
             */
            flattenTree: function(depth, current, sortFunction) {
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
                    retVal = retVal.concat(utils.flattenTree(depth + 1, current.children[i], sortFunction));
                }
                return retVal;
            },


            /*
             * Manage hosts
             * TODO - move host management into a separate service
             */
            refreshHosts: function($scope, resourcesFactory, cacheHosts, extraCallback) {
                // defend against empty scope
                if ($scope.hosts === undefined) {
                    $scope.hosts = {};
                }

                resourcesFactory.get_hosts(cacheHosts, function(allHosts) {
                    // This is a Map(Id -> Host)
                    $scope.hosts.mapped = allHosts;

                    // Get array of all hosts
                    $scope.hosts.all = utils.map_to_array(allHosts);

                    // This method gets called more than once. We don't watch to keep
                    // setting watches if we've already got one.
                    if ($scope.pools === undefined || $scope.pools.mapped === undefined) {
                        // Transfer path from pool to host
                        $scope.$watch('pools.mapped', function() {
                            utils.fix_pool_paths($scope);
                        });
                    } else {
                        utils.fix_pool_paths($scope);
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
            },

            refreshRunningForHost: function($scope, resourcesFactory, hostId) {
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
            },
            // add pool path to host
            fix_pool_paths: function($scope) {
                if ($scope.pools && $scope.pools.mapped && $scope.hosts && $scope.hosts.all) {
                    for(var i=0; i < $scope.hosts.all.length; i++) {
                        var host = $scope.hosts.all[i];
                        host.fullPath = $scope.pools.mapped[host.PoolID].fullPath;
                    }
                }
            },


            /*
             * Functions for setting up grid views
             * TODO - create angular controller for grids
             */
            buildTable: function(sort, headers) {
                var sort_icons = {};
                for(var i=0; i < headers.length; i++) {
                    sort_icons[headers[i].id] = (sort === headers[i].id?
                        'glyphicon-chevron-up' : 'glyphicon-chevron-down');
                }

                return {
                    sort: sort,
                    headers: headers,
                    sort_icons: sort_icons,
                    set_order: utils.set_order,
                    get_order_class: utils.get_order_class,
                };
            },

            set_order: function(order, table) {
                // Reset the icon for the last order
                if(DEBUG){
                    console.log('Resetting ' + table.sort + ' to down.');
                }
                table.sort_icons[table.sort] = 'glyphicon-chevron-down';

                if (table.sort === order) {
                    table.sort = "-" + order;
                    table.sort_icons[table.sort] = 'glyphicon-chevron-down';
                    if(DEBUG){
                        console.log('Sorting by -' + order);
                    }
                } else {
                    table.sort = order;
                    table.sort_icons[table.sort] = 'glyphicon-chevron-up';
                    if(DEBUG){
                        console.log('Sorting ' + table +' by ' + order);
                    }
                }
            },

            get_order_class: function(order, table) {
                return'glyphicon btn-link sort pull-right ' + table.sort_icons[order] +
                    ((table.sort === order || table.sort === '-' + order) ? ' active' : '');
            },


            /*
             * Helper and utility functions
             */
            map_to_array: function(data) {
                var arr = [];
                for (var key in data) {
                    arr[arr.length] = data[key];
                }
                return arr;
            },

            // TODO - use angular $location object
            unauthorized: function() {
                // FIXME: Need to injecct window.location to make this testable
                console.error('You don\'t appear to be logged in.');
                // show the login page and then refresh so we lose any incorrect state. CC-279
                window.location.href = "/#/login";
                window.location.reload();
            },

            indentClass: function(depth) {
                return 'indent' + (depth -1);
            },

            downloadFile: function(url){
                window.location = url;
            },

            getModeFromFilename: function(filename){
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
            },

            updateLanguage: function updateLanguage($scope, $cookies, $translate) {
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
        };

        return utils;
    }]);
})();
