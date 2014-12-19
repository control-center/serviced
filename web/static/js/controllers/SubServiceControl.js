function SubServiceControl($scope, $q, $routeParams, $location, resourcesService, authService, $serviceHealth, $modalService, $translate, $notification, $timeout){
    // Ensure logged in
    authService.checkLogin($scope);
    $scope.name = "servicedetails";
    $scope.params = $routeParams;
    $scope.servicesService = resourcesService;

    $scope.defaultHostAlias = location.hostname;
    var re = /\b(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b/;
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
        { id: 'Startup', name: 'label_service_startup' }
    ]);

    $scope.vhosts = buildTable('Name', [
        { id: 'Name', name: 'vhost_name'},
        { id: 'Application', name: 'vhost_application'},
        { id: 'ServiceEndpoint', name: 'vhost_service_endpoint'},
        { id: 'Name', name: 'vhost_url'}
    ]);

    $scope.ips = buildTable('ServiceName', [
        { id: 'ServiceName', name: 'tbl_virtual_ip_service'},
        { id: 'AssignmentType', name: 'tbl_virtual_ip_assignment_type'},
        { id: 'HostName', name: 'tbl_virtual_ip_host'},
        { id: 'PoolID', name: 'tbl_virtual_ip_pool'},
        { id: 'IPAddr', name: 'tbl_virtual_ip'}
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
        $modalService.create({
            templateUrl: "add-vhost.html",
            model: $scope,
            title: "add_virtual_host",
            actions: [
                {
                    role: "cancel",
                    action: function(){
                        $scope.vhosts.add = {};
                        this.close();
                    }
                },{
                    role: "ok",
                    label: "add_virtual_host",
                    action: function(){
                        if(this.validate()){
                            // disable ok button, and store the re-enable function
                            var enableSubmit = this.disableSubmitButton();

                            $scope.addVHost()
                                .success(function(data, status){
                                    this.close(); 
                                }.bind(this))
                                .error(function(data, status){
                                    this.createNotification("Unable to add virtual hosts", data.Detail).error(); 
                                    enableSubmit();
                                }.bind(this));
                        }
                    }
                }
            ],
            validate: function(){
                var name = $scope.vhosts.add.name;
                
                // if no name
                if(!name || !name.length){
                    this.createNotification("Unabled to add Virtual Host", "Missing name").error();
                    return false;
                }

                // if no services to bind to
                if(!$scope.vhosts.options.length){
                    this.createNotification("Unable to add Virtual Host", "No available application and service").error();
                    return false;
                }
                
                // if name already exists
                for (var i in $scope.vhosts.data) {
                    if (name == $scope.vhosts.data[i].Name) {
                        this.createNotification("Unabled to add Virtual Host", "Name already exists: "+ $scope.vhosts.add.name).error();
                        return false;
                    }
                }

                // if no endpoint selected
                if(!$scope.vhosts.add.app_ep){
                    this.createNotification("Unable to add Virtual Host", "No endpoint selected").error();
                    return false;
                }

                // if invalid characters
                var re = /^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$/;
                if(!re.test(name)){
                    this.createNotification("", $translate.instant("vhost_name_invalid") + " " + $scope.vhosts.add.name).error();
                    return false;
                }

                return true;
            }
        });
    };

    $scope.addVHost = function() {
        var name = $scope.vhosts.add.name;
        var serviceId = $scope.vhosts.add.app_ep.ServiceID;
        var serviceEndpoint = $scope.vhosts.add.app_ep.ServiceEndpoint;
        return resourcesService.add_vhost( serviceId, serviceEndpoint, name)
            .success(function(data, status){
                $notification.create("Added virtual host", data.Detail).success();
                $scope.vhosts.add = {};
                refreshServices($scope, resourcesService, false);
            });
    };

    // modalAssignIP opens a modal view to assign an ip address to a service
    $scope.modalAssignIP = function(ip, poolID) {
      $scope.ips.assign = {'ip':ip, 'value':null};
      resourcesService.get_pool_ips(poolID, function(data) {
        var options= [{'Value':'Automatic', 'IPAddr':null}];

        //host ips
        if ( data && data.HostIPs) {
          for(var i = 0; i < data.HostIPs.length; ++i) {
            var IPAddr = data.HostIPs[i].IPAddress;
            var value = 'Host: ' + IPAddr + ' - ' + data.HostIPs[i].InterfaceName;
            options.push({'Value': value, 'IPAddr':IPAddr});
            // set the default value to the currently assigned value
            if ($scope.ips.assign.ip.IPAddr == IPAddr) {
              $scope.ips.assign.value = options[ options.length-1];
            }
          }
        }

        //host ips
        if ( data && data.VirtualIPs) {
          for(var i = 0; i < data.VirtualIPs.length; ++i) {
            var IPAddr = data.VirtualIPs[i].IP;
            var value =  "Virtual IP: " + IPAddr;
            options.push({'Value': value, 'IPAddr':IPAddr});
            // set the default value to the currently assigned value
            if ($scope.ips.assign.ip.IPAddr == IPAddr) {
              $scope.ips.assign.value = options[ options.length-1];
            }
          }
        }

        //default to automatic
        if(!$scope.ips.assign.value) {
          $scope.ips.assign.value = options[0];
        }

        $scope.ips.assign.options = options;

        $modalService.create({
            templateUrl: "assign-ip.html",
            model: $scope,
            title: "assign_ip",
            actions: [
                {
                    role: "cancel"
                },{
                    role: "ok",
                    label: "assign_ip",
                    action: function(){
                        if(this.validate()){
                            // disable ok button, and store the re-enable function
                            var enableSubmit = this.disableSubmitButton();

                            $scope.assignIP()
                                .success(function(data, status){
                                    this.close();
                                }.bind(this))
                                .error(function(data, status){
                                    this.createNotification("Unable to Assign IP", data.Detail).error();
                                    enableSubmit();
                                }.bind(this));
                        }
                    }
                }
            ]
        });
      });
    };

    $scope.anyServicesExported = function(service) {
        if(service){
            for (var i in service.Endpoints) {
                if (service.Endpoints[i].Purpose == "export") {
                    return true;
                }
            }
            for (var i in service.children) {
                if ($scope.anyServicesExported(service.children[i])) {
                    return true;
                }
            }
        }
        return false;
    };


    $scope.assignIP = function() {
        var serviceID = $scope.ips.assign.ip.ServiceID;
        var IP = $scope.ips.assign.value.IPAddr;
        return resourcesService.assign_ip(serviceID, IP)
            .success(function(data, status){
                refreshServices($scope, resourcesService, false);
                $notification.create("Added IP", data.Detail).success();
            });
    };

    $scope.vhost_url = function(vhost) {
        var port = location.port === "" ? "" : ":"+location.port;
        var host = vhost.indexOf('.') === -1 ? vhost + "." + $scope.defaultHostAlias : vhost;
        return location.protocol + "//" + host + port
    };

    $scope.indent = function(depth){
        return {'padding-left': (20*depth) + "px"};
    };

    $scope.clickRunning = function(app, status, servicesService){
        toggleRunning(app, status, servicesService);
        $serviceHealth.update(app.ID);
    };

    function capitalizeFirst(str){
        return str.slice(0,1).toUpperCase() + str.slice(1);
    }

    $scope.clickRunningApp = function(app, status, servicesService) {

        // if this service has children and startup command, ask the user
        // if we should start service + children, or just service
        if(app.children && app.children.length && app.Startup){
            var displayStatus = capitalizeFirst(status),
                children = app.children || [],
                childCount = 0;

            // count number of descendent services that will start
            childCount = children.reduce(function countTheKids(acc, service){

                // if manual service, do not increment and
                // do not count children
                if(service.Launch === "manual"){
                    return acc;
                }

                acc++;

                // if no children, return
                if(!service.children){
                    return acc;

                // else, count children
                } else {
                    return service.children.reduce(countTheKids, acc);
                }
            }, 0);

            $modalService.create({
                template: ["<h4>"+ $translate.instant("choose_services_"+ status) +"</h4><ul>",
                    "<li>"+ $translate.instant(status +"_service_name", {name: "<strong>"+app.Name+"</strong>"}) +"</li>",
                    "<li>"+ $translate.instant(status +"_service_name_and_children", {name: "<strong>"+app.Name+"</strong>", count: "<strong>"+childCount+"</strong>"}) +"</li></ul>"
                ].join(""),
                model: $scope,
                title: $translate.instant(status +"_service"),
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        classes: " ",
                        label: $translate.instant(status +"_service"),
                        action: function(){
                            // the 4th arg here explicitly prevents child services
                            // from being started
                            toggleRunning(app, status, servicesService, true);
                            this.close();
                        }
                    },{
                        role: "ok",
                        label: $translate.instant(status +"_service_and_children", {count: childCount}),
                        action: function(){
                            toggleRunning(app, status, servicesService);
                            this.close();
                        }
                    }
                ]
            });

        // this service has no children or no startup command,
        // so start it the usual way
        } else {
            $scope.clickRunning(app, status, servicesService);
        }

    };

    $scope.clickEditContext = function(app, servicesService) {
	//set editor options for context editing
	$scope.codemirrorOpts = {
	    lineNumbers: true,
	    mode: "properties"
	}
	
        $scope.editableContext = makeEditableContext($scope.services.current.Context);

        $modalService.create({
            templateUrl: "edit-context.html",
            model: $scope,
            title: $translate.instant("edit_context"),
            actions: [
                {
                    role: "cancel"
                },{
                    role: "ok",
                    label: $translate.instant("btn_save_changes"),
                    action: function(){
                        // disable ok button, and store the re-enable function
                        var enableSubmit = this.disableSubmitButton();

                        saveContext(app, servicesService)
                            .success(function(data, status){
                                this.close(); 
                            }.bind(this))
                            .error(function(data, status){
                                this.createNotification("Updating service failed", data.Detail).error();
                                enableSubmit();
                            }.bind(this));
                    }
                }
            ],
	    onShow: function(){
		$scope.codemirrorRefresh = true;	
	    },
	    onHide: function(){
                $scope.codemirrorRefresh = false;
	    }
        });
    };

    function makeEditableContext(context){
        var editableContext = "";
        for(var key in context){
            editableContext += key + " " + context[key] + "\n";
        }
	if(!editableContext){ editableContext = ""; }
        return editableContext;
    }

    function saveContext(){
        //turn editableContext into a JSON object
        var lines = $scope.editableContext.split("\n");
        var context = {};
        for (var i=0; i<lines.length; ++i){
            var line = lines[i];
            if(line !== ""){
                var breakIndex = line.indexOf(' ');
                if(breakIndex !== -1){
                    var key = line.substr(0, breakIndex);
                    var value = line.substr(breakIndex+1);
                    context[key] = value;
                }else{
                    context[line] = "";
                }
            }
        }

        $scope.services.current.Context = context;
        return $scope.updateService();
    }

    $scope.clickRemoveVirtualHost = function(vhost) {
        $modalService.create({
            template: $translate.instant("confirm_remove_virtual_host") + " <strong>"+ vhost.Name +"</strong>",
            model: $scope,
            title: "remove_virtual_host",
            actions: [
                {
                    role: "cancel"
                },{
                    role: "ok",
                    label: "remove_virtual_host",
                    classes: "btn-danger",
                    action: function(){
                        resourcesService.delete_vhost( vhost.ApplicationId, vhost.ServiceEndpoint, vhost.Name, function( data) {
                            refreshServices($scope, resourcesService, false);
                        });
                        this.close();
                    }
                }
            ]
        });
    };

    $scope.editConfig = function(service, config) {
	$scope.editService = $.extend({}, service);
        $scope.editService.config = config;
        //set editor options for context editing
	$scope.codemirrorOpts = {
	    lineNumbers: true,
	    mode: getModeFromFilename($scope.editService.config)
	};
        $modalService.create({
            templateUrl: "edit-config.html",
            model: $scope,
            title: $translate.instant("title_edit_config") +" - "+ $scope.editService.config,
            bigModal: true,
            actions: [
                {
                    role: "cancel"
                },{
                    role: "ok",
                    label: "save",
                    action: function(){
                        if(this.validate()){
                            // disable ok button, and store the re-enable function
                            var enableSubmit = this.disableSubmitButton();

                            $scope.updateService()
                                .success(function(data, status){
                                    this.close(); 
                                }.bind(this))
                                .error(function(data, status){
                                    this.createNotification("Updating service failed", data.Detail).error();
                                    enableSubmit();
                                }.bind(this));
                        }
                    }
                }
            ],
            validate: function(){
                // TODO - actually validate
                return true;
            },
	    onShow: function(){
                $scope.codemirrorRefresh = true;
	    },
	    onHide: function(){
	        $scope.codemirrorRefresh = false;
	    }
        });
    };

    $scope.viewLog = function(serviceState) {
        $scope.editService = $.extend({}, serviceState);

	    resourcesService.get_service_state_logs(serviceState.ServiceID, serviceState.ID, function(log) {
            $scope.editService.log = log.Detail;
            $modalService.create({
                templateUrl: "view-log.html",
                model: $scope,
                title: "title_log",
                bigModal: true,
                actions: [
                    {
                        role: "cancel",
                        label: "close"
                    },{
                        classes: "btn-primary",
                        label: "download",
                        action: function(){
                            downloadFile('/services/' + serviceState.ServiceID + '/' + serviceState.ID + '/logs/download');
                        },
                        icon: "glyphicon-download"
                    }
                ],
                onShow: function(){
                    var textarea = this.$el.find("textarea");
                    textarea.scrollTop(textarea[0].scrollHeight - textarea.height());
                }
            });
        });
    };

    $scope.snapshotService = function(service) {
        resourcesService.snapshot_service(service.ID, function(label) {
            console.log('Snapshotted service name:%s label:%s', service.Name, label.Detail);
            // TODO: add the snapshot label to some partial view in the UI
        });
    };


    $scope.validateService = function() {
      // TODO: Validate name and startup command
      var svc = $scope.services.current,
          max = svc.InstanceLimits.Max,
          min = svc.InstanceLimits.Min,
          num = svc.Instances;
      if (typeof num == 'undefined' || (max > 0 && num > max) || (min > 0 && num < min)) {
        var msg = $translate.instant("instances_invalid") + " ";
        if (min > 0) {
          msg += $translate.instant("minimum") + " " + min;
          if (max > 0) {
            msg += ", "
          }
        }
        if (max > 0) {
          msg += $translate.instant("maximum") + " " + max;
        }
        $notification.create("", msg).error();
        return false;
      }
      return true;
    };

    $scope.updateService = function() {
        if ($scope.validateService()) {
            var serviceId = $scope.services.current.ID;

            return resourcesService.update_service($scope.services.current.ID, $scope.services.current)
                .success(function(data, status){
                    $notification.create("Updated service", serviceId).success();
                    refreshServices($scope, resourcesService, false);
                })
                .error(function(data, status){
                    $notification.create("Update service failed", data.Detail).error();
                });
        }
    };

    // Update the running instances so it is reflected when we save the changes
    function updateRunning() {
        if ($scope.params.serviceId) {
            refreshRunningForService($scope, resourcesService, $scope.params.serviceId, function() {
                wait.running = true;
                mashHostsToInstances();
            });
        }
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
                    crumb.url = '#/services/' + lineage[i].ID;
                }
                $scope.breadcrumbs.push(crumb);
            }
        }

        loadSubServiceHosts();
        $serviceHealth.update();
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
        resourcesService.kill_running(app.HostID, app.ID, function() {
            refreshRunningForService($scope, resourcesService, $scope.params.serviceId, function() {
                wait.running = true;
                mashHostsToInstances();
            });
        });
    };

    // restart all running instances for this service
    $scope.killRunningInstances = function(app){
        resourcesService.restart_service(app.ID, angular.noop);
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
            $modalService.create({
                templateUrl: "add-service.html",
                model: $scope,
                title: "add_service",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "add_service",
                        action: function(){
                            if(this.validate()){
                                $scope.add_service();
                                // NOTE: should wait for success before closing
                                this.close();
                            }
                        }
                    }
                ]
            });
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

    $scope.canChangeInstanceCount = function(min, max){
        // if min and max are both undefined,
        // this field should not be disabled
        if(min === undefined && max === undefined){
            return false;

        // if min and max are equal, this field
        // should be disabled
        } else {
            return min === max;
        }
    };


    $scope.$on("$destroy", function(){
        resourcesService.unregisterAllPolls();
    });

    function loadSubServiceHosts(){
        // to pull host data for running services, we need to make seperate "running" requests for each subservice
        // and add the host data to the subservice. We do this synchronously using promises here.

        var runningServiceDeferred = $q.defer();
        var runningServicePromise = runningServiceDeferred.promise;
        var ctr = 0;
        for(idx in $scope.services.subservices){
            (function(ctr){
                runningServicePromise.then(function(){
                    var deferred = $q.defer();
                    resourcesService.get_running_services_for_service($scope.services.subservices[ctr].ID, function(runningServices) {
                        $scope.services.subservices[ctr].runningHosts = [];

                        for (var i in runningServices) {
                            var instance = runningServices[i];
                            $scope.services.subservices[ctr].runningHosts.push({"ID": instance.HostID, "HostName": $scope.hosts.mapped[instance.HostID].Name});
                        }

                        deferred.resolve();
                    });
                });
            }(idx));
        }

        runningServiceDeferred.resolve();


        resourcesService.registerPoll("serviceHealth", $serviceHealth.update, 3000);
        resourcesService.registerPoll("running", updateRunning, 3000);
    }

    $scope.toggleChildren = function($event, app){
        var $e = $($event.target);
        $e.is(".glyphicon-chevron-down") ? hideChildren(app) : showChildren(app);
    };

    //we need to bring this function into scope so we can use ng-hide if an object is empty
    $scope.isEmptyObject = function(obj){
        return angular.equals({}, obj);
    };

    $scope.editCurrentService = function(){

        // clone service for editing
        $scope.editableService = $.extend({}, $scope.services.current);
        
        $modalService.create({
            templateUrl: "edit-service.html",
            model: $scope,
            title: "title_edit_service",
            actions: [
                {
                    role: "cancel"
                },{
                    role: "ok",
                    label: "btn_save_changes",
                    action: function(){
                        if(this.validate()){

                            // disable ok button, and store the re-enable function
                            var enableSubmit = this.disableSubmitButton();

                            // update service with recently edited service
                            resourcesService.update_service($scope.editableService.ID, $scope.editableService)
                                .success(function(data, status){
                                    $notification.create("Updated service", $scope.editableService.ID).success();
                                    refreshServices($scope, resourcesService, false);
                                    this.editableService = {};
                                    this.close();
                                }.bind(this))
                                .error(function(data, status){
                                    this.createNotification("Update service failed", data.Detail).error();
                                    enableSubmit();
                                }.bind(this));
                        }
                    }
                }
            ]
        });
    };

    function hideChildren(app){
        if(app.children){
            for(var i=0; i<app.children.length; ++i){
                var child = app.children[i];
                $("tr[data-id='" + child.ID + "'] td").hide();
                if(child.children !== undefined){
                    hideChildren(child);
                }
            }
        }

        //update icons
        $e = $("tr[data-id='"+app.ID+"'] td .glyphicon-chevron-down");
        $e.removeClass("glyphicon-chevron-down");
        $e.addClass("glyphicon-chevron-right");
    }

    function showChildren(app){
        if(app.children){
            for(var i=0; i<app.children.length; ++i){
                var child = app.children[i];
                $("tr[data-id='" + child.ID + "'] td").show();
                if(child.children !== undefined){
                    showChildren(child);
                }
            }
        }

        //update icons
        $e = $("tr[data-id='"+app.ID+"'] td .glyphicon-chevron-right");
        $e.removeClass("glyphicon-chevron-right");
        $e.addClass("glyphicon-chevron-down");
    }
    
    // Ensure we have a list of pools
    refreshPools($scope, resourcesService, false);
}
