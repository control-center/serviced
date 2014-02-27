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
angular.module('controlplane', ['ngRoute', 'ngCookies','ngDragDrop','pascalprecht.translate']).
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
            when('/servicesmap', {
                templateUrl: '/static/partials/view-service-map.html',
                controller: ServicesMapControl}).
            when('/hosts/:hostId', {
                templateUrl: '/static/partials/view-host-details.html',
                controller: HostDetailsControl
            }).
            when('/devmode', {
                templateUrl: '/static/partials/view-devmode.html',
                controller: DevControl
            }).
            otherwise({redirectTo: '/entry'});
    },]).
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
    $scope.brand_label = "brand_zcp";
    $scope.page_content = "entry_content";
    $scope.mainlinks = [
        { url: '#/apps', label: 'nav_apps' },
        { url: '#/hosts', label: 'nav_hosts' }
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

function DeployWizard($scope, resourcesService) {
    $scope.name='wizard';

    var validTemplateSelected = function() {
        return $scope.selectedTemplates().length > 0 && $scope.install.deploymentId.length > 0;
    };

    $scope.steps = [
/*        { content: '/static/partials/wizard-modal-1.html', label: 'label_step_select_hosts' }, */
        {
            content: '/static/partials/wizard-modal-2.html',
            label: 'label_step_select_app',
            validate: validTemplateSelected
        },
        { content: '/static/partials/wizard-modal-3.html', label: 'label_step_select_pool' },
        { content: '/static/partials/wizard-modal-4.html', label: 'label_step_deploy' }
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

    resourcesService.get_app_templates(false, function(templatesMap) {
        var templates = [];
        for (var key in templatesMap) {
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

            resourcesService.deploy_app_template({
                PoolId: $scope.install.selected.pool,
                TemplateId: selected[i].Id,
                DeploymentId: $scope.install.deploymentId
            }, function(result) {
                refreshServices($scope, resourcesService, false);
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
        { Name: 'Hostname D', IpAddr: '192.168.33.129', Id: 'DCDD3F0' }
    ];
    $scope.no_detected_hosts = ($scope.detected_hosts.length < 1);

    resetStepPage();

    // Get a list of pools (cached is OK)
    refreshPools($scope, resourcesService, true);
}

function DeployedAppsControl($scope, $routeParams, $location, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);
    $scope.name = "apps";
    $scope.params = $routeParams;
    $scope.servicesService = resourcesService;

    $scope.breadcrumbs = [
        { label: 'breadcrumb_deployed', itemClass: 'active' }
    ];

    $scope.secondarynav = [
        { label: 'nav_servicesmap', path: '/servicesmap' }
    ];

    $scope.services = buildTable('PoolId', [
        { id: 'Name', name: 'deployed_tbl_name'},
        { id: 'Deployment', name: 'deployed_tbl_deployment'},
        { id: 'PoolId', name: 'deployed_tbl_pool'},
        { id: 'Id', name: 'deployed_tbl_deployment_id'},
        { id: 'DesiredState', name: 'deployed_tbl_state' }
    ]);

    $scope.click_app = function(id) {
        $location.path('/services/' + id);
    };
    $scope.modalAddApp = function() {
        $('#addApp').modal('show');
    };

    $scope.clickRunning = toggleRunning;

    // Get a list of deployed apps
    refreshServices($scope, resourcesService, false);

    var setupNewService = function() {
        $scope.newService = {
            PoolId: 'default',
            ParentServiceId: '',
            DesiredState: 1,
            Launch: 'auto',
            Instances: 1,
            Description: '',
            ImageId: ''
        };
    };
    $scope.click_secondary = function(navlink) {
        if (navlink.path) {
            $location.path(navlink.path);
        }
        else if (navlink.modal) {
            $(navlink.modal).modal('show');
        }
        else {
            console.log('Unexpected navlink: %s', JSON.stringify(navlink));
        }
    };
    if ($scope.dev) {
        setupNewService();
        $scope.add_service = function() {
            resourcesService.add_service($scope.newService, function() {
                refreshServices($scope, resourcesService, false);
                setupNewService();
            });
        };
        $scope.secondarynav.push({ label: 'btn_add_service', modal: '#addService' });
    }
}

function SubServiceControl($scope, $routeParams, $location, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);
    $scope.name = "servicedetails";
    $scope.params = $routeParams;
    $scope.servicesService = resourcesService;

    $scope.breadcrumbs = [
        { label: 'breadcrumb_deployed', url: '#/apps' }
    ];

    $scope.services = buildTable('Name', [
        { id: 'Name', name: 'deployed_tbl_name'},
        { id: 'DesiredState', name: 'deployed_tbl_state' },
        { id: 'Details', name: 'deployed_tbl_details' }
    ]);

    $scope.click_app = function(id) {
        $location.path('/services/' + id);
    };

    $scope.indent = indentClass;
    $scope.clickRunning = toggleRunning;

    $scope.viewConfig = function(service) {
        $scope.editService = $.extend({}, service);
        $scope.editService.config = 'TODO: Implement';
        $('#editConfig').modal('show');
    };
    
    $scope.editConfig = function(service, config) {
        $scope.editService = $.extend({}, service);
        $scope.editService.config = config;
        $('#editConfig').modal('show');
    };
    
    $scope.viewLog = function(serviceState) {
        $scope.editService = $.extend({}, serviceState);
        resourcesService.get_service_state_logs(serviceState.ServiceId, serviceState.Id, function(log) {
            $scope.editService.log = log.Detail;
            $('#viewLog').modal('show');
        });
    };

    $scope.snapshotService = function(service) {
        resourcesService.snapshot_service(service.Id, function(label) {
            console.log('Snapshotted service name:%s label:%s', service.Name, label.Detail);
            // TODO: add the snapshot label to some partial view in the UI
        });
    };

    $scope.updateService = function() {
        resourcesService.update_service($scope.services.current.Id, $scope.services.current, function() {
            console.log('Updated %s', $scope.services.current.Id);
            var lastCrumb = $scope.breadcrumbs[$scope.breadcrumbs.length - 1];
            lastCrumb.label = $scope.services.current.Name;
        });
    }

    // Get a list of deployed apps
    refreshServices($scope, resourcesService, true, function() {
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

    var wait = { hosts: false, running: false };
    var mashHostsToInstances = function() {
        if (!wait.hosts || !wait.running) return;

        for (var i=0; i < $scope.running.data.length; i++) {
            var instance = $scope.running.data[i];
            instance.hostName = $scope.hosts.mapped[instance.HostId].Name;
        }
    }
    refreshHosts($scope, resourcesService, true, function() {
        wait.hosts = true;
        mashHostsToInstances();
    });
    refreshRunningForService($scope, resourcesService, $scope.params.serviceId, function() {
        wait.running = true;
        mashHostsToInstances();
    });

    $scope.killRunning = function(app) {
        resourcesService.kill_running(app.HostId, app.Id, function() {
            refreshRunningForService($scope, resourcesService, $scope.params.serviceId, function() {
                wait.running = true;
                mashHostsToInstances();
            });
        });
    };

    $scope.startTerminal = function(app) {
      window.open("http://" + window.location.hostname + ":50000")
    };

    var setupNewService = function() {
        $scope.newService = {
            PoolId: 'default',
            ParentServiceId: $scope.params.serviceId,
            DesiredState: 1,
            Launch: 'auto',
            Instances: 1,
            Description: '',
            ImageId: ''
        };
    };

    if ($scope.dev) {
        setupNewService();
        $scope.add_service = function() {
            resourcesService.add_service($scope.newService, function() {
                refreshServices($scope, resourcesService, false);
                setupNewService();
            });
        };
        $scope.showAddService = function() {
            $('#addService').modal('show');
        };
        $scope.deleteService = function() {
            var parent = $scope.services.current.ParentServiceId;
            console.log('Parent: %s, Length: %d', parent, parent.length);
            resourcesService.remove_service($scope.params.serviceId, function() {
                refreshServices($scope, resourcesService, false, function() {
                    if (parent && parent.length > 0) {
                        $location.path('/services/' + parent);
                    } else {
                        $location.path('/apps');
                    }
                });
            });
        };
    }
}

function HostsControl($scope, $routeParams, $location, $filter, $timeout,
                      resourcesService, authService)
{
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "hosts";
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
            refreshHosts($scope, resourcesService, false, hostCallback);
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

    $scope.modalAddHost = function() {
        $('#addHost').modal('show');
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
            refreshHosts($scope, resourcesService, false, hostCallback);
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
    refreshHosts($scope, resourcesService, false, hostCallback);
}

function HostDetailsControl($scope, $routeParams, $location, resourcesService, authService, statsService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "hostdetails";
    $scope.params = $routeParams;

    $scope.visualization = zenoss.visualization;
    $scope.visualization.url = 'http://' + $location.host() + ':8787';
    $scope.visualization.urlPath = '/metrics/static/performance/query/';
    $scope.visualization.urlPerformance = '/metrics/api/performance/query/';
    $scope.visualization.debug = false;

    $scope.breadcrumbs = [
        { label: 'breadcrumb_hosts', url: '#/hosts' }
    ];

    // Also ensure we have a list of hosts
    refreshHosts($scope, resourcesService, true);

    $scope.running = buildTable('Name', [
        { id: 'Name', name: 'running_tbl_running' },
        { id: 'StartedAt', name: 'running_tbl_start' },
        { id: 'View', name: 'running_tbl_actions' }
    ]);

    $scope.graph = buildTable('Name', [
        { id: 'CPU', name: 'graph_tbl_cpu'},
        { id: 'Memory', name: 'graph_tbl_mem'}
    ]);

    $scope.viewConfig = function(running) {
        $scope.editService = $.extend({}, running);
        $scope.editService.config = 'TODO: Implement';
        $('#editConfig').modal('show');
    };

    $scope.viewLog = function(running) {
        $scope.editService = $.extend({}, running);
        resourcesService.get_service_state_logs(running.ServiceId, running.Id, function(log) {
            $scope.editService.log = log.Detail;
            $('#viewLog').modal('show');
        });
    };

    $scope.click_app = function(instance) {
        $location.path('/services/' + instance.ServiceId);
    };

    $scope.killRunning = function(running) {
        resourcesService.kill_running(running.HostId, running.Id, function() {
            refreshRunningForHost($scope, resourcesService, $scope.params.hostId);
        });
    };

    refreshRunningForHost($scope, resourcesService, $scope.params.hostId);
    refreshHosts($scope, resourcesService, true, function() {
        if ($scope.hosts.current) {
            $scope.breadcrumbs.push({ label: $scope.hosts.current.Name, itemClass: 'active' });
        }
    });

    statsService.is_collecting(function(status) {
        if (status == 200) {
            $scope.collectingStats = true;
        } else {
            $scope.collectingStats = false;
        }
    });

    $scope.cpuconfig = {
        "datapoints": [
            {
                "aggregator": "avg",
                "color": "#aec7e8",
                "expression": null,
                "fill": false,
                "format": "%6.2f",
                "id": "system",
                "legend": "CPU (System)",
                "metric": "system",
                "name": "CPU (System)",
                "rate": true,
                "rateOptions": {},
                "type": "line"
            },
            {
                "aggregator": "avg",
                "color": "#98df8a",
                "expression": null,
                "fill": false,
                "format": "%6.2f",
                "id": "user",
                "legend": "CPU (User)",
                "metric": "user",
                "name": "CPU (User)",
                "rate": true,
                "rateOptions": {},
                "type": "line"
            }
        ],
        "downsample": "5m-avg",
        "footer": false,
        "format": "%6.2f",
        "maxy": null,
        "miny": 0,
        "range": {
            "end": "0s-ago",
            "start": "2d-ago"
        },
        "returnset": "EXACT",
        "tags": {},
        "type": "line"
    };

    $scope.memconfig = {
        "datapoints": [
            {
                "aggregator": "avg",
                "color": "#aec7e8",
                "expression": null,
                "fill": false,
                "format": "%6.2f",
                "id": "pgfault",
                "legend": "Page Faults",
                "metric": "pgfault",
                "name": "Page Faults",
                "rate": true,
                "rateOptions": {},
                "type": "line"
            }
        ],
        "downsample": "5m-avg",
        "footer": false,
        "format": "%6.2f",
        "maxy": null,
        "miny": 0,
        "range": {
            "end": "0s-ago",
            "start": "2d-ago"
        },
        "returnset": "EXACT",
        "tags": {},
        "type": "line"
    };
    
    $scope.rssconfig = {
        "datapoints": [
            {
                "aggregator": "avg",                
                "expression": null,
                "fill": false,
                "format": "%6.2f",
                "id": "pgfault",
                "legend": "RSS Memory",
                "metric": "rss",
                "name": "RSS Memory",                
                "rateOptions": {},
                "type": "line"
            }
        ],
        "downsample": "5m-avg",
        "footer": false,
        "format": "%6.2f",
        "maxy": null,
        "miny": 0,
        "range": {
            "end": "0s-ago",
            "start": "2d-ago"
        },
        "returnset": "EXACT",
        "tags": {},
        "type": "line"
    };

    $scope.drawn = {};

    $scope.viz = function(id, config) {
        if (!$scope.drawn[id]) {
            if (window.zenoss === undefined) {
                return "Not collecting stats, graphs unavailable";
            } else {
                zenoss.visualization.chart.create(id, config);
                $scope.drawn[id] = true;                
            }            
        }
    };
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
    var wait = { pools: false, hosts: false };
    var addHostsToPools = function() {
        if (!wait.pools || !wait.hosts) {
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
    refreshPools($scope, resourcesService, false, function() {
        wait.pools = true;
        addHostsToPools();
    });
    refreshHosts($scope, resourcesService, false, function() {
        wait.hosts = true;
        addHostsToPools();
    });
}

function ServicesMapControl($scope, $location, $routeParams, authService, resourcesService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "servicesmap";
    $scope.params = $routeParams;

    $scope.breadcrumbs = [
        { label: 'breadcrumb_deployed', url: '#/apps' },
        { label: 'breadcrumb_services_map', itemClass: 'active' }
    ];

    var data_received = {
        hosts: false,
        running: false,
        services: false
    };
    var nodeClasses = {};
    var runningServices = null;

    var draw = function() {
        if (!data_received.hosts) {
            console.log('Waiting for host data');
            return;
        }
        if (!data_received.running) {
            console.log('Waiting for running data');
            return;
        }
        if (!data_received.services) {
            console.log('Waiting for services data');
            return;
        }

        var states = [];
        var edges = [];

        for (var key in $scope.services.mapped) {
            var service = $scope.services.mapped[key];
            states[states.length] = {
                id: service.Id,
                value: { label: service.Name}
            };
            nodeClasses[service.Id] = 'service notrunning';

            if (service.ParentServiceId !== '') {
                var parent = $scope.services.mapped[service.ParentServiceId];
                nodeClasses[service.ParentServiceId] = 'service meta';
                edges[edges.length] = {
                    u: service.ParentServiceId,
                    v: key
                };
            }
        }

        var addedHosts = {};

        for (var i=0; i < runningServices.length; i++) {
            var running = runningServices[i];
            if (!addedHosts[running.HostId]) {
                states[states.length] = {
                    id: running.HostId,
                    value: { label: $scope.hosts.mapped[running.HostId].Name }
                };
                nodeClasses[running.HostId] = 'host';
                addedHosts[running.HostId] = true;
            }
            nodeClasses[running.ServiceId] = 'service';
            edges[edges.length] = {
                u: running.ServiceId,
                v: running.HostId
            };

        }

        var layout = dagreD3.layout().nodeSep(5).rankDir("LR")
        var renderer = new dagreD3.Renderer().layout(layout);
        var oldDrawNode = renderer.drawNode();
        renderer.drawNode(function(graph, u, svg) {
            oldDrawNode(graph, u, svg);
            svg.attr("class", "node " + nodeClasses[u]);
        });

        renderer.run(
            dagreD3.json.decode(states, edges),
            d3.select("svg g"));

        // Add zoom behavior
        var svg = d3.select("svg");
        svg.call(d3.behavior.zoom().on("zoom", function() {
            var ev = d3.event;
            svg.select("g")
                .attr("transform", "translate(" + ev.translate + ") scale(" + ev.scale + ")");
        }));
    };

    /*
     * Each successful resourceServices call will execute draw(),
     * but draw() will do an early return unless all required
     * data is available.
     */

    resourcesService.get_running_services(function(data) {
        data_received.running = true;
        runningServices = data;
        draw();
    });

    refreshHosts($scope, resourcesService, true, function() {
        data_received.hosts = true;
        draw();
    });

    refreshServices($scope, resourcesService, true, function() {
        data_received.services = true;
        draw();
    });
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
function NavbarControl($scope, $http, $cookies, $location, $route, $translate, authService) {
    $scope.name = 'navbar';
    $scope.brand = { url: '#/entry', label: 'brand_cp' };

    $scope.navlinks = [
        { url: '#/apps', label: 'nav_apps',
          sublinks: [ '#/services/', '#/servicesmap' ]
        },
        { url: '#/hosts', label: 'nav_hosts',
          sublinks: [ '#/hosts/', '#/hostsmap' ]
        }
    ];

    for (var i=0; i < $scope.navlinks.length; i++) {
        var cls = '';
        var currUrl = '#' + $location.path();
        if ($scope.navlinks[i].url === currUrl) {
            cls = 'active';
        } else {
            for (var j=0; j < $scope.navlinks[i].sublinks.length; j++) {
                if (currUrl.indexOf($scope.navlinks[i].sublinks[j]) === 0) {
                    cls = 'active';
                }
            }
        }
        $scope.navlinks[i].itemClass = cls;
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

    $scope.modalUserDetails = function() {
        $('#userDetails').modal('show');
    };
    updateLanguage($scope, $cookies, $translate);

    var helpMap = {
        '/static/partials/main.html': 'main.html',
        '/static/partials/login.html': 'login.html',
        '/static/partials/view-subservices.html': 'subservices.html',
        '/static/partials/view-apps.html': 'apps.html',
        '/static/partials/view-hosts.html': 'hosts.html',
        '/static/partials/view-host-map.html': 'hostmap.html',
        '/static/partials/view-service-map.html': 'servicemap.html',
        '/static/partials/view-host-details.html': 'hostdetails.html',
        '/static/partials/view-devmode.html': 'devmode.html'
    };

    $scope.help = {
        url: function() {
            return '/static/help/' + $scope.user.language + '/' + helpMap[$route.current.templateUrl];
        }
    };

}

function LanguageControl($scope, $cookies, $translate) {
    $scope.name = 'language';
    $scope.setUserLanguage = function() {
        console.log('User clicked %s', $scope.user.language);
        $cookies.Language = $scope.user.language;
        updateLanguage($scope, $cookies, $translate);
        $('#userDetails').modal('hide');
    };
    $scope.getLanguageClass = function(language) {
        return ($scope.user.language === language)? 'btn btn-primary active' : 'btn btn-primary';
    }
}

function updateLanguage($scope, $cookies, $translate) {
    var ln = 'en_US';
    if ($cookies.Language === undefined) {
//        console.log('Defaulting language to en_US');
    } else {
        ln = $cookies.Language;
//        console.log('Found language: %s', ln);
    }
    if ($scope.user) {
        $scope.user.language = ln;
    }
    $translate.uses(ln);
}

function DevControl($scope, $cookieStore, authService) {
    authService.checkLogin($scope);
    $scope.name = "developercontrol";

    var updateDevMode = function() {
        if ($scope.devmode.enabled) {
            $scope.devmode.enabledClass = 'btn btn-success active';
            $scope.devmode.enabledText = 'Enabled';
            $scope.devmode.disabledClass = 'btn btn-default off';
            $scope.devmode.disabledText = '\xA0'; // &nbsp;
        } else {
            $scope.devmode.enabledClass = 'btn btn-default off';
            $scope.devmode.enabledText = '\xA0';
            $scope.devmode.disabledClass = 'btn btn-danger active';
            $scope.devmode.disabledText = 'Disabled'; // &nbsp;
        }
    };
    $scope.devmode = {
        enabled: $cookieStore.get('ZDevMode')
    };
    $scope.setDevMode = function(enabled) {
        $scope.devmode.enabled = enabled;
        $cookieStore.put('ZDevMode', enabled);
        updateDevMode();
    };
    updateDevMode();
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
         * Get the list of services instances currently running for a given service.
         *
         * @param {string} serviceId The ID of the service to retrieve running instances for.
         * @param {function} callback Running services are passed to callback on success.
         */
        get_running_services_for_service: function(serviceId, callback) {
            $http.get('/services/' + serviceId + '/running').
                success(function(data, status) {
                    console.log('Got running services for %s', serviceId);
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
         * Get the list of services currently running on a particular host.
         *
         * @param {string} hostId The ID of the host to retrieve running services for.
         * @param {function} callback Running services are passed to callback on success.
         */
        get_running_services_for_host: function(hostId, callback) {
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
                    console.log('Terminating instance failed: %s', JSON.stringify(data));
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
                    console.log('Unable to retrieve service logs: %s', JSON.stringify(data));
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
                    console.log('Unable to retrieve service logs: %s', JSON.stringify(data));
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
                    console.log('Adding service failed: %s', JSON.stringify(data));
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
                    console.log('Updating service failed: %s', JSON.stringify(data));
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
                    console.log('Deploying app template failed: %s', JSON.stringify(data));
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
                    console.log('Snapshot service failed: %s', JSON.stringify(data));
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
                    console.log('Removing service failed: %s', JSON.stringify(data));
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
                    console.log('serviced is not collecting stats');
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

