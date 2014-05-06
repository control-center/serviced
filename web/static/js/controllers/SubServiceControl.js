function SubServiceControl($scope, $routeParams, $location, $interval, resourcesService, authService) {
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
        { id: 'Startup', name: 'label_service_startup' }
    ]);

    $scope.vhosts = buildTable('vhost_name', [
        { id: 'Name', name: 'vhost_name'},
        { id: 'Application', name: 'vhost_application'},
        { id: 'ServiceEndpoint', name: 'vhost_service_endpoint'},
        { id: 'URL', name: 'vhost_url'},
        { id: 'Action', name: 'vhost_actions'},
    ]);

    //add vhost data (includes name, app & service endpoint)
    $scope.vhosts.add = {};

    //app & service endpoint option for adding a new virtual host
    $scope.vhosts.options = [];

    $scope.click_app = function(id) {
        $location.path('/services/' + id);
    };

    $scope.modalAddVHost = function() {
        $('#addVHost').modal('show');
    };

    $scope.addVHost = function() {
        if (!$scope.vhosts.add.name || $scope.vhosts.add.name.length <= 0) {
            console.error( "Cannot add vhost -- missing name");
            return;
        }

        if ($scope.vhosts.options.length <= 0) {
            console.error( "Cannot add vhost -- no available application and service");
            return;
        }

        var name = $scope.vhosts.add.name;
        var serviceId = $scope.vhosts.add.app_ep.ServiceId;
        var serviceEndpoint = $scope.vhosts.add.app_ep.ServiceEndpoint;
        resourcesService.add_vhost( serviceId, serviceEndpoint, name, function() {
            $scope.vhosts.add = {};
            refreshServices($scope, resourcesService, false);
        });
    };

    $scope.vhost_url = function( vhost) {
        return get_vhost_url( $location, vhost);
    }

    $scope.indent = indentClass;
    $scope.clickRunning = toggleRunning;

    $scope.viewConfig = function(service) {
        $scope.editService = $.extend({}, service);
        $scope.editService.config = 'TODO: Implement';
        $('#editConfig').modal('show');
    };

    $scope.clickRemoveVirtualHost = function(vhost) {
        resourcesService.delete_vhost( vhost.ApplicationId, vhost.ServiceEndpoint, vhost.Name, function( data) {
            refreshServices($scope, resourcesService, false);
        });
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
    };
    // Update the running instances so it is reflected when we save the changes
    //TODO: Destroy/cancel this interval when we are not on the subservices page, or get rid of it all together
    function updateRunning() {
        if ($scope.params.serviceId) {
            refreshRunningForService($scope, resourcesService, $scope.params.serviceId, function() {
                wait.running = true;
                mashHostsToInstances();
            });
        }
    }
    $interval(updateRunning, 3000);
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
    };
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
        window.open("http://" + window.location.hostname + ":50000");
    };

    var setupNewService = function() {
        $scope.newService = {
            poolID: 'default',
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

    console.log($scope);
}