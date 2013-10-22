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
angular.module('controlplane', ['ngCookies','ngDragDrop']).
    config(['$routeProvider', function($routeProvider) {
        $routeProvider.
            when('/entry', { 
                templateUrl: '/static/partials/main.html',
                controller: EntryControl}).
            when('/login', {
                templateUrl: '/static/partials/login.html',
                controller: LoginControl}).
            when('/services/:serviceId', {
                templateUrl: '/static/partials/view-subservices.html',
                controller: SubServiceControl}).
            when('/apps', {
                templateUrl: '/static/partials/view-apps.html',
                controller: DeployedAppsControl
            }).
            when('/hosts', {
                templateUrl: '/static/partials/view-hosts.html',
                controller: HostsControl}).
            when('/hostsmap', {
                templateUrl: '/static/partials/view-host-map.html',
                controller: HostsMapControl}).
            when('/hosts/:hostId', {
                templateUrl: '/static/partials/view-host-details.html',
                controller: HostDetailsControl
            }).
            otherwise({redirectTo: '/entry'});
    }]).
    factory('resourcesService', ResourcesService).
    factory('servicesService', ServicesService).
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

function EntryControl($scope, authService) {
    authService.checkLogin($scope);
    $scope.brand_label = "Zenoss Control Plane";
    $scope.page_content = "You can install Resource Manager, Analytics, and Impact here."; 
    $scope.mainlinks = [
        { url: '#/apps', label: 'Applications' },
        { url: '#/hosts', label: 'Hosts' }
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
                authService.login(true, $scope.username);
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

function DeployWizard($scope, resourcesService, servicesService) {
    var validTemplateSelected = function() {
        return $scope.selectedTemplates().length > 0;
    };

    $scope.steps = [
/*        { content: '/static/partials/wizard-modal-1.html', label: 'Select Hosts' }, */
        {
            content: '/static/partials/wizard-modal-2.html',
            label: 'Select Applications',
            validate: validTemplateSelected
        },
        { content: '/static/partials/wizard-modal-3.html', label: 'Select Resource Pool' },
        { content: '/static/partials/wizard-modal-4.html', label: 'Start / Go' },
    ];

    $scope.install = { 
        selected: {
            pool: 'default'
        },
        templateClass: function(template) {
            var cls = "block-data control-group";
            if (template.depends) {
                cls += " indented";
            }
            return cls;
        },
        templateSelected: function(template) {
            if (template.depends) {
                $scope.install.selected[template.depends] = true;
            }
        },
        templateDisabled: function(template) {
            if (template.disabledBy) {
                return $scope.install.selected[template.disabledBy];
            }
            return false;
        },
        templateSelectedFormDiv: function() {
            return (!nextClicked || validTemplateSelected())?
                '':'has-error';
        }
    };
    var nextClicked = false;

    servicesService.get_app_templates(false, function(templatesMap) {
        var templates = [];
        for (key in templatesMap) {
            var template = templatesMap[key];
            template.Id = key;
            templates[templates.length] = template;
        }
        $scope.install.templateData = templates;
    });

    $scope.selectedTemplates = function() {
        var templates = [];
        for (var i=0; i < $scope.install.templateData.length; i++) {
            var template = $scope.install.templateData[i];
            if ($scope.install.selected[template.Id]) {
                templates[templates.length] = template;
            }
        }
        return templates;
    };

    var step = 0;
    var resetStepPage = function() {
        step = 0;
        $scope.step_page = $scope.steps[step].content;
    };

    $scope.addHostStart = function() {
        $scope.newHost = {};
        $scope.step_page = '/static/partials/wizard-modal-addhost.html';
    };

    $scope.addHostCancel = function() {
        $scope.step_page = $scope.steps[step].content;
    }

    $scope.addHostFinish = function() {
        $scope.newHost.Name = $scope.newHost.IpAddr;
        $scope.newHost.Id = 'fakefakefake';
        $scope.newHost.selected = true;
        $scope.detected_hosts.push($scope.newHost);
        $scope.step_page = $scope.steps[step].content;
    };

    $scope.hasPrevious = function() {
        return step > 0 && 
            ($scope.step_page === $scope.steps[step].content);
    };

    $scope.hasNext = function() {
        return (step + 1) < $scope.steps.length && 
            ($scope.step_page === $scope.steps[step].content);
    };

    $scope.hasFinish = function() {
        return (step + 1) === $scope.steps.length;
    };

    $scope.step_item = function(index) {
        var cls = index <= step ? 'active' : 'inactive';
        if (index === step) { 
            cls += ' current';
        }
        return cls;
    };

    $scope.step_label = function(index) {
        return index < step ? 'done' : '';
    };

    $scope.wizard_next = function() {
        nextClicked = true;
        if ($scope.step_page !== $scope.steps[step].content) {
            $scope.step_page = $scope.steps[step].content;
            return;
        }
        if ($scope.steps[step].validate) {
            if (!$scope.steps[step].validate()) {
                return;
            }
        }
        step += 1;
        $scope.step_page = $scope.steps[step].content;
        nextClicked = false;
    };

    $scope.wizard_previous = function() {
        step -= 1;
        $scope.step_page = $scope.steps[step].content;
    };
    
    $scope.wizard_finish = function() {
        var selected = $scope.selectedTemplates();
        var f = true;
        var dName = "";
        for (var i=0; i < selected.length; i++) {
            if (f) {
                f = false;
            } else {
                dName += ", ";
                if (i + 1 === selected.length) {
                    dName += "and ";
                }
            }
            dName += selected[i].Name;

            servicesService.deploy_app_template({ 
                PoolId: $scope.install.selected.pool,
                TemplateId: selected[i].Id
            }, function(result) {
                refreshServices($scope, servicesService, false);
            });
        }

        $scope.services.deployed = {
            name: dName,
            multi: (selected.length > 1),
            class: "deployed alert alert-success",
            show: true,
            url: "http://localhost:8080/",
            deployment: "ready"
        };
        $('#addApp').modal('hide');
        resetStepPage();
    };

    $scope.detected_hosts = [
        { Name: 'Hostname A', IpAddr: '192.168.34.1', Id: 'A071BF1' },
        { Name: 'Hostname B', IpAddr: '192.168.34.25', Id: 'B770DAD' },
        { Name: 'Hostname C', IpAddr: '192.168.33.99', Id: 'CCD090B' },
        { Name: 'Hostname D', IpAddr: '192.168.33.129', Id: 'DCDD3F0' },
    ];
    $scope.no_detected_hosts = ($scope.detected_hosts.length < 1);

    resetStepPage();

    // Get a list of pools (cached is OK)
    refreshPools($scope, resourcesService, true);
}

function DeployedAppsControl($scope, $routeParams, $location, servicesService, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);
    $scope.name = "resources";
    $scope.params = $routeParams;
    $scope.servicesService = servicesService;

    $scope.breadcrumbs = [
        { label: 'Deployed Apps', itemClass: 'active' }
    ];


    $scope.services = buildTable('Deployment', [
        { id: 'Name', name: 'Application'}, 
        { id: 'Deployment', name: 'Deployment Status'},
        { id: 'PoolId', name: 'Resource Pool'},
        { id: 'DesiredState', name: 'Status' }
    ]);

    $scope.click_app = function(id) {
        $location.path('/services/' + id);
    };

    $scope.clickRunning = toggleRunning;

    // Get a list of deployed apps
    refreshServices($scope, servicesService, false);
}

function fakeConfig() {
    console.log('TODO: Replace this function');
    return '#\n' +
           '# Ethernet frame types\n' +
           '#               This file describes some of the various Ethernet\n' +
           '#               protocol types that are used on Ethernet networks.\n' +
           '#\n' +
           '# This list could be found on:\n' +
           '#         http://www.iana.org/assignments/ethernet-numbers\n' +
           '#\n' +
           '# <name>    <hexnumber> <alias1>...<alias35> #Comment\n' +
           '#\n' +
           'IPv4            0800    ip ip4          # Internet IP (IPv4)\n' +
           'X25             0805\n' +
           'ARP             0806    ether-arp       #\n' +
           'FR_ARP          0808                    # Frame Relay ARP        [RFC1701]\n' +
           'BPQ             08FF                    # G8BPQ AX.25 Ethernet Packet\n' +
           'DEC             6000                    # DEC Assigned proto\n' +
           'DNA_DL          6001                    # DEC DNA Dump/Load\n' +
           'DNA_RC          6002                    # DEC DNA Remote Console\n' +
           'DNA_RT          6003                    # DEC DNA Routing\n' +
           'LAT             6004                    # DEC LAT\N' +
           'DIAG            6005                    # DEC Diagnostics\n' +
           'CUST            6006                    # DEC Customer use\n' +
           'SCA             6007                    # DEC Systems Comms Arch\n' +
           'TEB             6558                    # Trans Ether Bridging   [RFC1701]\n' +
           'RAW_FR          6559                    # Raw Frame Relay        [RFC1701]\n' +
           'AARP            80F3                    # Appletalk AARP\n' +
           'ATALK           809B                    # Appletalk\n' +
           '802_1Q          8100    8021q 1q 802.1q dot1q # 802.1Q Virtual LAN tagged frame\n' +
           'IPX             8137                    # Novell IPX\n' +
           'NetBEUI         8191                    # NetBEUI\n' +
           'IPv6            86DD    ip6             # IP version 6\n' +
           'PPP             880B                    # PPP\N' +
           'ATMMPOA         884C                    # MultiProtocol over ATM\n' +
           'PPP_DISC        8863                    # PPPoE discovery messages\n' +
           'PPP_SES         8864                    # PPPoE session messages\n' +
           'ATMFATE         8884                    # Frame-based ATM Transport over Ethernet\n' +
           'LOOP            9000    loopback        # loop proto\n';
}

function SubServiceControl($scope, $routeParams, $location, servicesService, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);
    $scope.name = "resources";
    $scope.params = $routeParams;
    $scope.servicesService = servicesService;

    $scope.breadcrumbs = [
        { label: 'Deployed Apps', url: '#/apps' }
    ];

    $scope.services = buildTable('Name', [
        { id: 'Name', name: 'Application'}, 
        { id: 'DesiredState', name: 'Status' },
        { id: 'Details', name: 'Details' }
    ]);

    $scope.click_app = function(id) {
        $location.path('/services/' + id);
    };

    $scope.indent = indentClass;
    $scope.clickRunning = toggleRunning;

    $scope.viewConfig = function(service) {
        $scope.editService = $.extend({}, service);
        $scope.editService.config = fakeConfig(); // FIXME
        $('#editConfig').modal('show');
    };

    $scope.viewLog = function(service) {
        $scope.editService = $.extend({}, service);
        servicesService.get_service_logs(service.Id, function(log) {
            $scope.editService.log = log.Detail;
            $('#viewLog').modal('show');
        });
    };

    $scope.updateService = function() {
        servicesService.update_service($scope.services.current.Id, $scope.services.current, function() {
            console.log('Updated %s', $scope.services.current.Id);
        });
    }

    // Get a list of deployed apps
    refreshServices($scope, servicesService, true, function() {
        if ($scope.services.current) {
            var lineage = getServiceLineage($scope.services.mapped, $scope.services.current);
            for (var i=0; i < lineage.length; i++) {
                var crumb = {
                    label: lineage[i].Name
                };
                if (i == lineage.length - 1) {
                    crumb.itemClass = 'active';
                } else {
                    crumb.url = '#/services/' + lineage[i].Id;
                }
                $scope.breadcrumbs.push(crumb);
            }
        }
    });
}

function HostsControl($scope, $routeParams, $location, $filter, $timeout, 
                      resourcesService, authService) 
{
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "resources";
    $scope.params = $routeParams;
    $scope.toggleCollapsed = function(toggled) {
        toggled.collapsed = !toggled.collapsed;
        if (toggled.children === undefined) {
            return;
        }
        toggled.icon = toggled.collapsed? POOL_ICON_CLOSED : POOL_ICON_OPEN;
        for (var i=0; i < toggled.children.length; i++) {
            toggleCollapse(toggled.children[i], toggled.collapsed);
        }
    };
    $scope.itemClass = itemClass;
    $scope.indent = indentClass;
    $scope.newPool = {};
    $scope.newHost = {};

    $scope.add_host = function() {
        resourcesService.add_host($scope.newHost, function(data) {
            // After adding, refresh our list
            refreshHosts($scope, resourcesService, false, false, hostCallback);
        });
        // Reset for another add
        $scope.newHost = {
            PoolId: $scope.params.poolId
        };
    };

    $scope.addSubpool = function(poolId) {
        $scope.newPool.ParentId = poolId;
        $('#addPool').modal('show');
    };
    $scope.delSubpool = function(poolId) {
        resourcesService.remove_pool(poolId, function() {
            refreshPools($scope, resourcesService, false);
        });
    };

    // Build metadata for displaying a list of pools
    $scope.pools = buildTable('Id', [
        { id: 'Id', name: 'Id'}, 
        { id: 'ParentId', name: 'Parent Id'},
        { id: 'Priority', name: 'Priority'}
    ])

    var clearLastStyle = function() {
        var lastPool = $scope.pools.mapped[$scope.selectedPool];
        if (lastPool) {
            lastPool.current = false;
        }
    };

    $scope.clearSelectedPool = function() {
        clearLastStyle();
        $scope.selectedPool = null;
        $scope.subPools = null;
        hostCallback();
    };

    $scope.clickHost = function(hostId) {
        $location.path('/hosts/' + hostId);
    };

    $scope.clickPool = function(poolId) {
        var topPool = $scope.pools.mapped[poolId];
        if (!topPool || $scope.selectedPool === poolId) {
            $scope.clearSelectedPool();
            return;
        }
        clearLastStyle();
        topPool.current = true;

        var allowed = {};
        addChildren(allowed, topPool);
        $scope.subPools = allowed;
        $scope.selectedPool = poolId;
        hostCallback();
    };

    $scope.dropped = [];

    $scope.filterHosts = function() {
        if (!$scope.hosts.filtered) {
            $scope.hosts.filtered = [];
        }
        // Run ordering filter, built in
        var ordered = $filter('orderBy')($scope.hosts.all, $scope.hosts.sort);
        // Run search filter, built in
        var filtered = $filter('filter')(ordered, $scope.hosts.search);
        // Run filter for pool and child pools, custom
        var treeFiltered = $filter('treeFilter')(filtered, 'PoolId', $scope.subPools);

        // As a side effect, save number of hosts before paging
        if (treeFiltered) {
            $scope.hosts.filteredCount = treeFiltered.length;
        } else {
            $scope.hosts.filteredCount = 0;
        }
        var page = $scope.hosts.page? $scope.hosts.page : 1;
        var pageSize = $scope.hosts.pageSize? $scope.hosts.pageSize : 5;
        var itemsToTake = page * pageSize;
        $scope.hosts.filteredCountLimit = itemsToTake;
        if (treeFiltered) {
            $scope.hosts.filtered = treeFiltered.splice(0, itemsToTake);
        }
        return $scope.hosts.filtered;
    };

    $scope.loadMore = function() {
        if ($scope.hosts.filteredCount && $scope.hosts.filteredCountLimit &&
           $scope.hosts.filteredCountLimit < $scope.hosts.filteredCount) {
            $scope.hosts.page += 1;
            $scope.filterHosts();
            return true;
        }

        return false;
    };

    $scope.dropIt = function(event, ui) {
        var poolId = $(event.target).attr('data-pool-id');
        var pool = $scope.pools.mapped[poolId];
        var host = $scope.dropped[0];

        if (poolId === host.PoolId) {
            // Nothing changed. Don't bother showing the dialog.
            return;
        }

        $scope.move = {
            host: host,
            newpool: poolId
        };
        $scope.dropped = [];
        $('#confirmMove').modal('show');
    };

    $scope.confirmMove = function() {
        console.log('Reassigning %s to %s', $scope.move.host.Name, $scope.move.newpool);
        var modifiedHost = $.extend({}, $scope.move.host);
        modifiedHost.PoolId = $scope.move.newpool;
        resourcesService.update_host(modifiedHost.Id, modifiedHost, function() {
            refreshHosts($scope, resourcesService, false, false, hostCallback);
        });
    };

    // Function for adding new pools
    $scope.add_pool = function() {
        console.log('Adding pool %s as child of pool %s', $scope.newPool.Id, $scope.params.poolId);
        resourcesService.add_pool($scope.newPool, function(data) {
            // After adding, refresh our list
            refreshPools($scope, resourcesService, false);
        });
        // Reset for another add
        $scope.newPool = {};
    };

    // Function for removing the current pool
    $scope.remove_pool = function() {
        console.log('Removing pool %s', $scope.params.poolId);
        resourcesService.remove_pool($scope.params.poolId, function(data) {
            refreshPools($scope, resourcesService, false);
        });
    };

    // Build metadata for displaying a list of hosts
    $scope.hosts = buildTable('Name', [
        { id: 'Name', name: 'Name'},
        { id: 'fullPath', name: 'Assigned Resource Pool'},
    ]);

    $scope.clickMenu = function(index) {
        $('#pool_menu_' + index).addClass('tempvis');
        setTimeout(function() {
            $('#pool_menu_' + index).removeClass('tempvis');
        }, 600);
    };

    var hostCallback = function() {
        $scope.hosts.page = 1;
        $scope.hosts.pageSize = 10;
        $scope.filterHosts();
        $timeout($scope.hosts.scroll, 100);
    };

    // Ensure we have a list of pools
    refreshPools($scope, resourcesService, false);
    // Also ensure we have a list of hosts
    refreshHosts($scope, resourcesService, false, false, hostCallback);
}

function HostDetailsControl($scope, $routeParams, $location, resourcesService, servicesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "hostdetails";
    $scope.params = $routeParams;

    $scope.breadcrumbs = [
        { label: 'Hosts', url: '#/hosts' }
    ];

    // Also ensure we have a list of hosts
    refreshHosts($scope, resourcesService, true, true);

    $scope.running = buildTable('Name', [
        { id: 'Name', name: 'Sub Applications' },
        { id: 'StartedAt', name: 'Start Time' },
        { id: 'View', name: 'View' }
    ]);

    $scope.viewConfig = function(running) {
        $scope.editService = $.extend({}, running);
        $scope.editService.config = fakeConfig(); // FIXME
        $('#editConfig').modal('show');
    };

    $scope.viewLog = function(running) {
        $scope.editService = $.extend({}, running);
        servicesService.get_service_state_logs(running.Id, function(log) {
            $scope.editService.log = log.Detail;
            $('#viewLog').modal('show');
        });
    };

    $scope.killRunning = killRunning;
    $scope.unkillRunning = unkillRunning;
    refreshRunning($scope, resourcesService, $scope.params.hostId);
    refreshHosts($scope, resourcesService, true, true, function() {
        if ($scope.hosts.current) {
            $scope.breadcrumbs.push({ label: $scope.hosts.current.Name, itemClass: 'active' });
        }
    })
}

function HostsMapControl($scope, $routeParams, $location, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "hostsmap";
    $scope.params = $routeParams;
    $scope.itemClass = itemClass;
    $scope.indent = indentClass;

    $scope.addSubpool = function(poolId) {
        $scope.newPool.ParentId = poolId;
        $('#addPool').modal('show');
    };
    $scope.delSubpool = function(poolId) {
        resourcesService.remove_pool(poolId, function() {
            refreshPools($scope, resourcesService, false);
        });
    };
    $scope.newPool = {};
    $scope.newHost = {};

    var clearLastStyle = function() {
        var lastPool = $scope.pools.mapped[$scope.selectedPool];
        if (lastPool) {
            lastPool.current = false;
        }
    };

    $scope.clearSelectedPool = function() {
        clearLastStyle();
        $scope.selectedPool = null;
        var root = { Id: 'All Resource Pools', children: $scope.pools.tree };
        $scope.hosts.filteredCount = $scope.hosts.all.length;
        selectNewRoot(root);
    };

    var countFromPool = function(e) {
        if (e.isHost) return 1;
        if (e.children === undefined) return 0;

        var count = 0;
        for (var i=0; i < e.children.length; i++) {
            count += countFromPool(e.children[i]);
        }
        return count;
    };

    $scope.clickPool = function(poolId) {
        var topPool = $scope.pools.mapped[poolId];
        if (!topPool || $scope.selectedPool === poolId) {
            $scope.clearSelectedPool();
            return;
        }
        clearLastStyle();
        topPool.current = true;

        $scope.selectedPool = poolId;
        $scope.hosts.filteredCount = countFromPool(topPool);
        selectNewRoot(topPool);
    };
    var width = 857;
    var height = 567;

    var cpuCores = function(h) { 
        return h.Cores;
    };
    var memoryCapacity = function(h) {
        return h.Memory;
    };
    var poolBgColor = function(p) {
        return p.isHost? null : color(p.Id);
    };
    var hostText = function(h) {
        return h.isHost? h.Name : null;
    }

    var color = d3.scale.category20c();
    var treemap = d3.layout.treemap()
        .size([width, height])
        .value(memoryCapacity);

    var position = function() {
        this.style("left", function(d) { return d.x + "px"; })
            .style("top", function(d) { return d.y + "px"; })
            .style("width", function(d) { return Math.max(0, d.dx - 1) + "px"; })
            .style("height", function(d) { return Math.max(0, d.dy - 1) + "px"; });
    };

    $scope.selectionButtonClass = function(id) {
        var cls = 'btn btn-link nav-link';
        if ($scope.treemapSelection === id) {
            cls += ' active';
        }
        return cls;
    };

    $scope.selectByMemory = function() {
        $scope.treemapSelection = 'memory';
        selectNewValue(memoryCapacity);
    };
    $scope.selectByCores = function() {
        $scope.treemapSelection = 'cpu';
        selectNewValue(cpuCores);
    };

    var selectNewValue = function(valFunc) {
        var node = d3.select("#hostmap").
            selectAll(".node").
            data(treemap.value(valFunc).nodes)
        node.enter().
            append("div").
            attr("class", "node");
        node.transition().duration(1000).
            call(position).
            style("background", poolBgColor).
            text(hostText);
        node.exit().remove();
    };

    var selectNewRoot = function(newroot) {
        console.log('Selected %s', newroot.Id);
        var node = d3.select("#hostmap").
            datum(newroot).
            selectAll(".node").
            data(treemap.nodes)

        node.enter().
            append("div").
            attr("class", "node");

        node.transition().duration(1000).
            call(position).
            style("background", poolBgColor).
            text(hostText);
        node.exit().remove();
    };

    var hostsAddedToPools = false;
    var addHostsToPools = function() {
        if (!$scope.pools.mapped || !$scope.hosts.mapped) {
            console.log('Need both pools and hosts');
            return;
        }
        if (hostsAddedToPools) {
            console.log('Already built');
            return;
        }

        console.log('Preparing tree map');
        $scope.hosts.filteredCount = $scope.hosts.all.length;
        hostsAddedToPools = true;
        for(var key in $scope.hosts.mapped) {
            var host = $scope.hosts.mapped[key];
            var pool = $scope.pools.mapped[host.PoolId];
            if (pool.children === undefined) {
                pool.children = [];
            }
            pool.children.push(host);
            host.isHost = true;
        }
        var root = { Id: 'All Resource Pools', children: $scope.pools.tree };
        selectNewRoot(root);
    };
    $scope.treemapSelection = 'memory';
    // Also ensure we have a list of hosts
    refreshPools($scope, resourcesService, false, addHostsToPools);
    refreshHosts($scope, resourcesService, false, false, addHostsToPools);
}

/*
 * Recurse through children building up allowed along the way.
 */
function addChildren(allowed, parent) {
    allowed[parent.Id] = true;
    if (parent.children) {
        for (var i=0; i < parent.children.length; i++) {
            addChildren(allowed, parent.children[i]);
        }
    }
}

// Controller for top nav
function NavbarControl($scope, $http, $cookies, $location, authService) {
    $scope.management = 'Management';
    $scope.configuration = 'Configuration';
    $scope.resources = 'Resources';
    $scope.username = $scope.username? $scope.username : $cookies['ZUsername'];
    $scope.brand = { url: '#/entry', label: 'Control Plane' };
    
    $scope.navlinks = [
        { url: '#/apps', label: 'Deployed Apps' },
        { url: '#/hosts', label: 'Hosts' }
    ];

    for (var i=0; i < $scope.navlinks.length; i++) {
        $scope.navlinks[i].itemClass = ($scope.navlinks[i].url === '#' + $location.path())? 
            "active" : "";
    }

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

function ServicesService($http, $location) {
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
                console.log('Unable to retrieve services');
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
                console.log('Unable to retrieve app templates');
                if (status === 401) {
                    unauthorized($location);
                }
            });
    };

    return {
        get_services: function(cacheOk, callback) {
            if (cacheOk && cached_services && cached_services_map) {
                console.log('Using cached services');
                callback(cached_services, cached_services_map);
            } else {
                _get_services_tree(callback);
            }
        },

        get_service_logs: function(serviceId, callback) {
            $http.get('/services/' + serviceId + '/logs').
                success(function(data, status) {
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Unable to retrieve service logs: %s', JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        },


        get_service_state_logs: function(serviceStateId, callback) {
            $http.get('/running/' + serviceStateId + '/logs').
                success(function(data, status) {
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Unable to retrieve service logs: %s', JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        },

        get_app_templates: function(cacheOk, callback) {
            if (cacheOk && cached_app_templates) {
                console.log('Using cached app templates');
                callback(cached_app_templates);
            } else {
                _get_app_templates(callback);
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
                    console.log('Adding service failed: %s', JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        },

        update_service: function(serviceId, editedService, callback) {
            $http.put('/services/' + serviceId, editedService).
                success(function(data, status) {
                    console.log('Updated service %s', serviceId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Updating service failed: %s', JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        },

        deploy_app_template: function(deployDef, callback) {
            $http.post('/templates/deploy', deployDef).
                success(function(data, status) {
                    console.log('Deployed app template');
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Deploying app template failed: %s', JSON.stringify(data));
                    if (status === 401) {
                        unauthorized($location);
                    }
                });
        },

        remove_service: function(serviceId, callback) {
            $http.delete('/services/' + serviceId).
                success(function(data, status) {
                    console.log('Removed service %s', serviceId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Removing service failed: %s', JSON.stringify(data));
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


        get_running_services: function(hostId, callback) {
            $http.get('/hosts/' + hostId + '/running').
                success(function(data, status) {
                    console.log('Got running services for %s', hostId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Unable to acquire running services: %s', JSON.stringify(data));
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
                    console.log('Adding pool failed: %s', JSON.stringify(data));
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
            $http.put('/pools/' + poolId, editedPool).
                success(function(data, status) {
                    console.log('Updated pool %s', poolId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Updating pool failed: %s', JSON.stringify(data));
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
                    console.log('Removed pool %s', poolId);
                    callback(data);
                }).
                error(function(data, status) {
                    console.log('Removing pool failed: %s', JSON.stringify(data));
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
                    console.log('Adding host failed: %s', JSON.stringify(data));
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
                    console.log('Updating host failed: %s', JSON.stringify(data));
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
                    console.log('Removing host failed: %s', JSON.stringify(data));
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
            if (loggedIn) {
                $scope.loggedIn = true;
                $scope.username = userName;
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
            }
        }
        if (extraCallback) {
            extraCallback();
        }
    });
}

function getFullPath(allPools, pool) {
    if (!allPools || !pool.ParentId || !allPools[pool.ParentId]) {
        return pool.Id;
    }
    return getFullPath(allPools, allPools[pool.ParentId]) + " > " + pool.Id;
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
                p.fullPath = p.Id;
            }
        }

        if ($scope.params && $scope.params.poolId) {
            $scope.pools.current = allPools[$scope.params.poolId];
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
    app.DesiredState = newState;
    servicesService.update_service(app.Id, app, function() {
        updateRunning(app);
    });
}

function killRunning(app) {
    app.DesiredState = 0;
    console.log("TODO: Kill service");
    updateRunning(app);
}

function unkillRunning(app) {
    app.DesiredState = 1;
    console.log("TODO: Remove this function");
    updateRunning(app);
}


function updateRunning(app) {
    if (app.DesiredState === 1) {
        app.runningText = "started";
        app.notRunningText = "\xA0"; // &nbsp
        app.runningClass = "btn btn-success active";
        app.notRunningClass = "btn btn-default off";
    } else if (app.DesiredState === -1) {
        app.runningText = "restarting";
        app.notRunningText = "\xA0"; // &nbsp
        app.runningClass = "btn btn-info active";
        app.notRunningClass = "btn btn-default off";
    } else {
        app.runningText = "\xA0"; // &nbsp
        app.notRunningText = "stopped";
        app.runningClass = "btn btn-default off";
        app.notRunningClass = "btn btn-danger active";
    }
    if (app.Deployment !== "successful") {
        app.runningClass += " disabled";
        app.notRunningClass += " disabled";
    }
}

function refreshHosts($scope, resourcesService, cacheHosts, cacheHostsPool, extraCallback) {
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

function refreshRunning($scope, resourcesService, hostId) {
    if ($scope.running === undefined) {
        $scope.running = {};
    }

    resourcesService.get_running_services(hostId, function(runningServices) {
        $scope.running.data = runningServices;
        for (var i=0; i < runningServices.length; i++) {
            runningServices[i].DesiredState = 1; // All should be running
            runningServices[i].Deployment = 'successful'; // TODO: Replace
            updateRunning(runningServices[i]);
        }
    });
}

function fix_pool_paths($scope) {
    if ($scope.pools && $scope.pools.mapped && $scope.hosts && $scope.hosts.all) {
        for(var i=0; i < $scope.hosts.all.length; i++) {
            var host = $scope.hosts.all[i];
            host.fullPath = $scope.pools.mapped[host.PoolId].fullPath;
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

