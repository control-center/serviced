


function SubServiceControl($scope, $routeParams, $location, $interval, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);
    $scope.name = "servicedetails";
    $scope.params = $routeParams;
    $scope.servicesService = resourcesService;

    $scope.defaultHostAlias = location.hostname;
    var re = /\b(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b/
    if (re.test(location.hostname) || location.hostname == "localhost") {
        $.getJSON("/hosts/defaultHostAlias", "", function(data) {
            $scope.defaultHostAlias = data.hostalias;
        });
    }

    $scope.breadcrumbs = [
        { label: 'breadcrumb_deployed', url: '#/apps' }
    ];

    $scope.services = buildTable('Name', [
        { id: 'Name', name: 'deployed_tbl_name'},
        { id: 'DesiredState', name: 'deployed_tbl_state' },
        { id: 'Health', name: 'Health' },
        { id: 'Startup', name: 'label_service_startup' }
    ]);

    $scope.vhosts = buildTable('vhost_name', [
        { id: 'Name', name: 'vhost_name'},
        { id: 'Application', name: 'vhost_application'},
        { id: 'ServiceEndpoint', name: 'vhost_service_endpoint'},
        { id: 'URL', name: 'vhost_url'},
        { id: 'Action', name: 'vhost_actions'},
    ]);

    $scope.ips = buildTable('Service', [
        { id: 'Service', name: 'tbl_virtual_ip_service'},
        { id: 'Application', name: 'tbl_virtual_ip_application'},
        { id: 'AssignmentType', name: 'tbl_virtual_ip_assignment_type'},
        { id: 'Host', name: 'tbl_virtual_ip_host'},
        { id: 'Pool', name: 'tbl_virtual_ip_pool'},
        { id: 'IPAddress', name: 'tbl_virtual_ip'},
        { id: 'Port', name: 'tbl_virtual_ip_port'},
        { id: 'Actions', name: 'tbl_virtual_ip_actions'}
    ]);

    //add vhost data (includes name, app & service endpoint)
    $scope.vhosts.add = {};

    //app & service endpoint option for adding a new virtual host
    $scope.vhosts.options = [];

    $scope.click_app = function(id) {
        $location.path('/services/' + id);
    };

    $scope.click_pool = function(id) {
        $location.path('/pools/' + id);
    };

    $scope.click_host = function(id) {
        $location.path('/hosts/' + id);
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
        var serviceId = $scope.vhosts.add.app_ep.ServiceID;
        var serviceEndpoint = $scope.vhosts.add.app_ep.ServiceEndpoint;
        resourcesService.add_vhost( serviceId, serviceEndpoint, name, function() {
            $scope.vhosts.add = {};
            refreshServices($scope, resourcesService, false);
        });
    };

    // modalAssignIP opens a modal view to assign an ip address to a service
    $scope.modalAssignIP = function(ip, poolID) {
      $scope.ips.assign = {'ip':ip, 'value':null}
      resourcesService.get_pool_ips(poolID, function(data) {
        var options= [{'Value':'Automatic', 'IPAddr':null}]

        //host ips
        if ( data && data.HostIPs) {
          for(var i = 0; i < data.HostIPs.length; ++i) {
            var IPAddr = data.HostIPs[i].IPAddress
            var value = 'Host: ' + IPAddr + ' - ' + data.HostIPs[i].InterfaceName
            options.push({'Value': value, 'IPAddr':IPAddr})
            // set the default value to the currently assigned value
            if ($scope.ips.assign.ip.IPAddr == IPAddr) {
              $scope.ips.assign.value = options[ options.length-1]
            }
          }
        }

        //host ips
        if ( data && data.VirtualIPs) {
          for(var i = 0; i < data.VirtualIPs.length; ++i) {
            var IPAddr = data.VirtualIPs[i].IP
            var value =  "Virtual IP: " + IPAddr
            options.push({'Value': value, 'IPAddr':IPAddr})
            // set the default value to the currently assigned value
            if ($scope.ips.assign.ip.IPAddr == IPAddr) {
              $scope.ips.assign.value = options[ options.length-1]
            }
          }
        }

        //default to automatic
        if(!$scope.ips.assign.value) {
          $scope.ips.assign.value = options[0]
        }

        $scope.ips.assign.options = options
        $('#assignIP').modal('show');
      })
    };

    $scope.AssignIP = function() {
        var serviceID = $scope.ips.assign.ip.ServiceID;
        var IP = $scope.ips.assign.value.IPAddr;
        resourcesService.assign_ip(serviceID, IP, function(data) {
            refreshServices($scope, resourcesService, false);
        });
    };

    $scope.vhost_url = function(vhost) {
        var port = location.port == "" ? "" : ":"+location.port;
        return location.protocol + "//" + vhost + "." + $scope.defaultHostAlias + port;
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
        resourcesService.get_service_state_logs(serviceState.ServiceID, serviceState.Id, function(log) {
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

    function updateHealth(ServiceID) {
        $.getJSON("/servicehealth", function(packet) {
            var healths = packet["Statuses"];
            var timestamp = packet["Timestamp"];
            for (var ServiceId in healths) {
                data = healths[ServiceId];
                element = document.getElementById("health-tooltip-" + ServiceId);
                if (element) {
                    element.title = "";
                    passingAny = false;
                    failingAny = false;
                    lateAny = false;
                    unknownAny = false;
                    for (var name in data) {
                        if (timestamp - data[name].Timestamp >= data[name].Interval * 2) {
                            data[name].Status = "unknown";
                        }
                        if (data[name].Status == "passed") {
                            passingAny = true;
                        } else if (data[name].Status == "failed") {
                            failingAny = true;
                        } else if (data[name].Status == "unknown") {
                            unknownAny = true;
                        }
                        element.title += name + ":" + data[name].Status + "\n";
                    }

                    var setStatus = function(status) {
                        document.getElementById("health-" + ServiceId).className = "healthIcon glyphicon glyphicon-" + status;
                    };

                    if (failingAny) {
                        setStatus("exclamation-sign bad");
                    } else if (!passingAny && unknownAny) {
                        setStatus("minus-sign unknown");
                    } else if (passingAny && unknownAny) {
                        setStatus("question-sign unknown");
                    } else if (passingAny && !unknownAny) {
                        setStatus("ok-sign good");
                    }
                }
            }
        });
    }

    // Update the running instances so it is reflected when we save the changes
    //TODO: Destroy/cancel this interval when we are not on the subservices page, or get rid of it all together
    function updateRunning() {
        if ($scope.params.serviceId) {
            refreshRunningForService($scope, resourcesService, $scope.params.serviceId, function() {
                wait.running = true;
                mashHostsToInstances();
            });
            updateHealth();
        }
    }
    if(!angular.isDefined($scope.updateRunningInterval)) {
        $scope.updateRunningInterval = $interval(updateRunning, 3000);
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
        updateHealth();
    });

    $scope.$on('$destroy', function() {
        $interval.cancel($scope.updateRunningInterval);
        $scope.updateRunningInterval = undefined;
    });

    var wait = { hosts: false, running: false };
    var mashHostsToInstances = function() {
        if (!wait.hosts || !wait.running) return;

        for (var i=0; i < $scope.running.data.length; i++) {
            var instance = $scope.running.data[i];
            instance.hostName = $scope.hosts.mapped[instance.HostID].Name;
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
        resourcesService.kill_running(app.HostID, app.Id, function() {
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
            ParentServiceID: $scope.params.serviceId,
            DesiredState: 1,
            Launch: 'auto',
            Instances: 1,
            Description: '',
            ImageID: ''
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
            var parent = $scope.services.current.ParentServiceID;
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
