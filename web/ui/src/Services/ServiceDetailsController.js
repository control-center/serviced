/* globals controlplane: true */

/* ServiceDetailsController
 * Displays details of a specific service
 */
(function() {
    'use strict';

    controlplane.controller("ServiceDetailsController",
    ["$scope", "$q", "$routeParams", "$location", "resourcesFactory",
    "authService", "$modalService", "$translate", "$notification",
    "$timeout", "servicesFactory", "miscUtils", "hostsFactory",
    "CCUIState", "$cookies",
    function($scope, $q, $routeParams, $location, resourcesFactory,
    authService, $modalService, $translate, $notification,
    $timeout, servicesFactory, utils, hostsFactory,
    CCUIState, $cookies){

        // Ensure logged in
        authService.checkLogin($scope);
        $scope.resourcesFactory = resourcesFactory;
        $scope.hostsFactory = hostsFactory;

        $scope.defaultHostAlias = $location.host();
        if(utils.needsHostAlias($location.host())){
            resourcesFactory.getHostAlias().success(function(data) {
                $scope.defaultHostAlias = data.hostalias;
            });
        }

        //add vhost data (includes name, app & service endpoint)
        $scope.vhosts = {
            add: {}
        };

        $scope.click_pool = function(id) {
            resourcesFactory.routeToPool(id);
        };

        $scope.click_host = function(id) {
            resourcesFactory.routeToHost(id);
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
                                        $notification.create("Added virtual host", data.Detail).success();
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
                    if(!$scope.vhosts.data.length){
                        this.createNotification("Unable to add Virtual Host", "No available application and service").error();
                        return false;
                    }

                    // if name already exists
                    for (var i in $scope.vhosts.data) {
                        if (name === $scope.vhosts.data[i].Name) {
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
            var serviceId = $scope.vhosts.add.app_ep.ApplicationId;
            var serviceEndpoint = $scope.vhosts.add.app_ep.ServiceEndpoint;
            return resourcesFactory.addVHost( serviceId, serviceEndpoint, name)
                .success(function(data, status){
                    $scope.vhosts.add = {};
                });
        };

        // modalAssignIP opens a modal view to assign an ip address to a service
        $scope.modalAssignIP = function(ip, poolID) {
          $scope.ips.assign = {'ip':ip, 'value':null};
          resourcesFactory.getPoolIPs(poolID)
              .success(function(data) {
                var options= [{'Value':'Automatic', 'IPAddr':null}];

                var i, IPAddr, value;
                //host ips
                if (data && data.HostIPs) {
                  for(i = 0; i < data.HostIPs.length; ++i) {
                    IPAddr = data.HostIPs[i].IPAddress;
                    value = 'Host: ' + IPAddr + ' - ' + data.HostIPs[i].InterfaceName;
                    options.push({'Value': value, 'IPAddr':IPAddr});
                    // set the default value to the currently assigned value
                    if ($scope.ips.assign.ip.IPAddr === IPAddr) {
                      $scope.ips.assign.value = options[ options.length-1];
                    }
                  }
                }

                //virtual ips
                if (data && data.VirtualIPs) {
                  for(i = 0; i < data.VirtualIPs.length; ++i) {
                    IPAddr = data.VirtualIPs[i].IP;
                    value =  "Virtual IP: " + IPAddr;
                    options.push({'Value': value, 'IPAddr':IPAddr});
                    // set the default value to the currently assigned value
                    if ($scope.ips.assign.ip.IPAddr === IPAddr) {
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
                                            $notification.create("Added IP", data.Detail).success();
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
              })
              .error((data, status) => {
                $notification.create("Unable to retrieve IPs", data.Detail).error();
              });
        };

        $scope.anyServicesExported = function(service) {
            if(service){
                for (var i in service.Endpoints) {
                    if (service.Endpoints[i].Purpose === "export") {
                        return true;
                    }
                }
                for (var j in service.children) {
                    if ($scope.anyServicesExported(service.children[j])) {
                        return true;
                    }
                }
            }
            return false;
        };


        $scope.assignIP = function() {
            var serviceID = $scope.ips.assign.ip.ServiceID;
            var IP = $scope.ips.assign.value.IPAddr;
            return resourcesFactory.assignIP(serviceID, IP)
                .success(function(data, status){
                    // HACK: update(true) forces a full update to
                    // work around issue https://jira.zenoss.com/browse/CC-939
                    servicesFactory.update(true);
                });
        };

        $scope.vhost_url = function(vhost) {
            var port = location.port === "" ? "" : ":"+location.port;
            var host = vhost.indexOf('.') === -1 ? vhost + "." + $scope.defaultHostAlias : vhost;
            return location.protocol + "//" + host + port;
        };

        $scope.indent = function(depth){
            return {'padding-left': (15*depth) + "px"};
        };

        // sets a service to start, stop or restart state
        $scope.setServiceState = function(service, state, skipChildren){
            service[state](skipChildren).error(function(data, status){
                $notification.create("Unable to " + state + " service", data.Detail).error();
            });
        };

        // filters to be used when counting how many descendent
        // services will be affected by a state change
        var serviceStateChangeFilters = {
            // only stopped services will be started
            "start": service => service.desiredState === 0,
            // only started services will be stopped
            "stop": service => service.desiredState === 1,
            // only started services will be restarted
            "restart": service => service.desiredState === 1
        };

        // clicks to a service's start, stop, or restart
        // button should first determine if the service has
        // children and ask the user to choose to start all
        // children or only the top service
        $scope.clickRunning = function(service, state){
            var filterFn = serviceStateChangeFilters[state];
            var childCount = utils.countTheKids(service, filterFn);

            // if the service has affected children, check if the user
            // wants to start just the service, or the service and children
            if(childCount > 0){
                $scope.modal_confirmSetServiceState(service, state, childCount);

            // if no children, just start the service
            } else {
                $scope.setServiceState(service, state);
            }
            servicesFactory.updateHealth();
        };

        // verifies if use wants to start parent service, or parent
        // and all children
        $scope.modal_confirmSetServiceState = function(service, state, childCount){
            $modalService.create({
                template: ["<h4>"+ $translate.instant("choose_services_"+ state) +"</h4><ul>",
                    "<li>"+ $translate.instant(state +"_service_name", {name: "<strong>"+service.name+"</strong>"}) +"</li>",
                    "<li>"+ $translate.instant(state +"_service_name_and_children", {name: "<strong>"+service.name+"</strong>", count: "<strong>"+childCount+"</strong>"}) +"</li></ul>"
                ].join(""),
                model: $scope,
                title: $translate.instant(state +"_service"),
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        classes: " ",
                        label: $translate.instant(state +"_service"),
                        action: function(){
                            // the arg here explicitly prevents child services
                            // from being started
                            $scope.setServiceState(service, state, true);
                            this.close();
                        }
                    },{
                        role: "ok",
                        label: $translate.instant(state +"_service_and_children", {count: childCount}),
                        action: function(){
                            $scope.setServiceState(service, state);
                            this.close();
                        }
                    }
                ]
            });
        };

        $scope.clickVHostEnable = function(vhost){
            resourcesFactory.enableVHost( vhost.ApplicationId, vhost.ServiceEndpoint, vhost.Name)
                .error((data, status) => {
                    $notification.create("Start Vhost failed", data.Detail).error();
                });
        };

    $scope.clickVHostDisable = function(vhost){
        resourcesFactory.disableVHost( vhost.ApplicationId, vhost.ServiceEndpoint, vhost.Name)
            .error((data, status) => {
                $notification.create("Stop Vhost failed", data.Detail).error();
            });

    };
        $scope.clickEditContext = function() {
            //set editor options for context editing
            $scope.codemirrorOpts = {
                lineNumbers: true,
                mode: "properties"
            };

            $scope.editableService = angular.copy($scope.services.current.model);
            $scope.editableContext = makeEditableContext($scope.editableService.Context);

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

                            $scope.editableService.Context = makeStorableContext($scope.editableContext);

                            $scope.updateService($scope.editableService)
                                .success(function(data, status){
                                    $notification.create("Updated service", $scope.editableService.ID).success();
                                    this.close();
                                }.bind(this))
                                .error(function(data, status){
                                    this.createNotification("Update service failed", data.Detail).error();
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
        function makeStorableContext(context){
            //turn editableContext into a JSON object
            var lines = context.split("\n"),
                storable = {};

            lines.forEach(function(line){
                var delimitIndex, key, val;

                if(line !== ""){
                    delimitIndex = line.indexOf(" ");
                    if(delimitIndex !== -1){
                        key = line.substr(0, delimitIndex);
                        val = line.substr(delimitIndex + 1);
                        storable[key] = val;
                    } else {
                        context[line] = "";
                    }
                }
            });

            return storable;
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
                            resourcesFactory.removeVHost( vhost.ApplicationId, vhost.ServiceEndpoint, vhost.Name)
                                .success(() => {
                                    servicesFactory.update();
                                    $notification.create("Removed VHost", vhost.Name).success();
                                })
                                .error((data, status) => {
                                    $notification.create("Remove VHost failed", data.Detail).error();
                                });

                            this.close();
                        }
                    }
                ]
            });
        };

        $scope.editConfig = function(config) {
            $scope.editableService = angular.copy($scope.services.current.model);
            $scope.selectedConfig = config;

            //set editor options for context editing
            $scope.codemirrorOpts = {
                lineNumbers: true,
                mode: utils.getModeFromFilename($scope.selectedConfig)
            };

            $modalService.create({
                templateUrl: "edit-config.html",
                model: $scope,
                title: $translate.instant("title_edit_config") +" - "+ $scope.selectedConfig,
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

                                $scope.updateService($scope.editableService)
                                    .success(function(data, status){
                                        $notification.create("Updated service", $scope.editableService.ID).success();
                                        this.close();
                                    }.bind(this))
                                    .error(function(data, status){
                                        this.createNotification("Update service failed", data.Detail).error();
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

        $scope.viewLog = function(instance) {
            $scope.editService = angular.copy(instance);

            resourcesFactory.getInstanceLogs(instance.model.ServiceID, instance.model.ID)
                .success(function(log) {
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
                                label: "refresh",
                                icon: "glyphicon-repeat",
                                action: function() {
                                    var textarea = this.$el.find("textarea");
                                    resourcesFactory.getInstanceLogs(instance.model.ServiceID, instance.id)
                                        .success(function(log) {
                                            $scope.editService.log = log.Detail;
                                            textarea.scrollTop(textarea[0].scrollHeight - textarea.height());
                                        })
                                        .error((data, status) => {
                                            this.createNotification("Unable to fetch logs", data.Detail).error();
                                        });
                                }
                            },{
                                classes: "btn-primary",
                                label: "download",
                                action: function(){
                                    utils.downloadFile('/services/' + instance.model.ServiceID + '/' + instance.model.ID + '/logs/download');
                                },
                                icon: "glyphicon-download"
                            }
                        ],
                        onShow: function(){
                            var textarea = this.$el.find("textarea");
                            textarea.scrollTop(textarea[0].scrollHeight - textarea.height());
                        }
                    });
                })
                .error(function(data, status){
                    $notification.create("Unable to fetch logs", data.Detail).error();
                });
        };

        $scope.validateService = function() {
          // TODO: Validate name and startup command
          var svc = $scope.services.current.model,
              max = svc.InstanceLimits.Max,
              min = svc.InstanceLimits.Min,
              num = svc.Instances;
          if (typeof num === 'undefined' || (max > 0 && num > max) || (min > 0 && num < min)) {
            var msg = $translate.instant("instances_invalid") + " ";
            if (min > 0) {
              msg += $translate.instant("minimum") + " " + min;
              if (max > 0) {
                msg += ", ";
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

        $scope.updateService = function(newService) {
            if ($scope.validateService()) {
                return resourcesFactory.updateService($scope.services.current.model.ID, newService)
                    .success((data, status) => {
                        servicesFactory.update();
                        this.editableService = {};
                    });
            }
        };

        $scope.subNavClick = function(crumb){
            if(crumb.id){
                $scope.routeToService(crumb.id);
            } else {
                // TODO - just call subnavs usual function
                $location.path(crumb.url);
            }
        };

        $scope.routeToService = function(id, e){
            // if an event is present, we may
            // need to prevent it from performing
            // default navigation behavior
            if(e){
                // ctrl click opens in new tab,
                // so allow that to happen and don't
                // bother routing the current view
                if(e.ctrlKey){
                    return;
                }

                // if middle click, don't update
                // current view
                if(e.button === 1){
                    return;
                }

                // otherwise, prevent default so
                // we can handle the view routing
                e.preventDefault();
            }

            $location.update_path("/services/"+id, true);
            $scope.params.serviceId = id;
            $scope.services.current = servicesFactory.get($scope.params.serviceId);
            $scope.update();
        };

        $scope.getService = function(id) {
            return servicesFactory.get(id);
        };

        $scope.isServiceRunning = function(id) {
            var service = servicesFactory.get(id);
            return service.desiredState === 1;
        };

        $scope.update = function(){
            if($scope.services.current){
                $scope.services.subservices = $scope.services.current.descendents;
                $scope.vhosts.data = $scope.services.current.hosts;
                $scope.ips.data = $scope.services.current.addresses;

                // update instances
                $scope.services.current.getServiceInstances();

                // setup breadcrumbs
                $scope.breadcrumbs = makeCrumbs($scope.services.current);

                // update serviceTreeState
                $scope.serviceTreeState = CCUIState.get($cookies.ZUsername, "serviceTreeState");

                // create an entry in tree state for the
                // current service
                if(!($scope.services.current.id in $scope.serviceTreeState)){
                    $scope.serviceTreeState[$scope.services.current.id] = {};

                    var treeState = $scope.serviceTreeState[$scope.services.current.id];

                    // create default entries from all descendents
                    $scope.services.current.descendents.forEach(descendent => {
                        // TODO - formalize this state object
                        treeState[descendent.id] = {
                            hidden: false,
                            collapsed: false
                        };
                    });
                }

                // property for view to bind for tree state
                $scope.services.currentTreeState = $scope.serviceTreeState[$scope.services.current.id];
            }

            servicesFactory.updateHealth();
        };

        // restart all running instances for this service
        $scope.killRunningInstances = function(app){
            resourcesFactory.restartService(app.ID)
                .error((data, status) => {
                    $notification.create("Stop Service failed", data.Detail).error();
                });
        };

        $scope.startTerminal = function(app) {
            window.open("http://" + window.location.hostname + ":50000");
        };



        $scope.getHostName = function(id){
            if(hostsFactory.get(id)){
                return hostsFactory.get(id).name;
            } else {
                // TODO - if unknown host, dont make linkable
                // and use custom css to show unknown
                return "unknown";
            }
        };

        // expand/collapse state of service tree nodes
        $scope.serviceTreeState = CCUIState.get($cookies.ZUsername, "serviceTreeState");
        // servicedTreeState is a collection of objects
        // describing if nodes in a service tree are hidden or collapsed.
        // It is first keyed by the id of the current service context (the
        // service who's name is at the top of the page), then keyed by
        // the service in question. eg:
        //
        // current service id
        //      -> child service id
        //          -> hidden
        //          -> collapsed
        //      -> child service id
        //          -> hidden
        //          -> collapsed
        //      ...

        $scope.toggleChildren = function(service){
            if(!$scope.services.current){
                console.warn("Cannot store toggle state: no current service");
                return;
            }

            // stored state for the current service's
            // service tree
            var treeState = $scope.services.currentTreeState;

            // if this service is marked as collapsed in
            // this particular tree view, show its children
            if(treeState[service.id].collapsed){
                treeState[service.id].collapsed = false;
                $scope.showChildren(service);

            // otherwise, hide its children
            } else {
                treeState[service.id].collapsed = true;
                $scope.hideChildren(service);
            }
        };

        $scope.hideChildren = function(service){
            // get the state of the current service's tree
            var treeState = $scope.services.currentTreeState;

            service.children.forEach(function(child){
                treeState[child.id].hidden = true;
                $scope.hideChildren(child);
            });
        };

        $scope.showChildren = function(service){
            var treeState = $scope.services.currentTreeState;

            service.children.forEach(function(child){
                treeState[child.id].hidden = false;

                // if this child service is not marked
                // as collapsed, show its children
                if(!treeState[child.id].collapsed){
                    $scope.showChildren(child);
                }
            });
        };

        //we need to bring this function into scope so we can use ng-hide if an object is empty
        $scope.isEmptyObject = function(obj){
            return angular.equals({}, obj);
        };

        $scope.isIsvc = function(service){
            return service.isIsvc();
        };

        $scope.hasCurrentInstances = function(){
            return $scope.services && $scope.services.current && $scope.services.current.hasInstances();
        };

        $scope.editCurrentService = function(){

            // clone service for editing
            $scope.editableService = angular.copy($scope.services.current.model);

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
                                $scope.updateService($scope.editableService)
                                    .success(function(data, status){
                                        $notification.create("Updated service", $scope.editableService.ID).success();
                                        this.close();
                                    }.bind(this))
                                    .error(function(data, status){
                                        this.createNotification("Update service failed", data.Detail).error();
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ],
                validate: function(){
                    if($scope.editableService.InstanceLimits.Min > $scope.editableService.Instances || $scope.editableService.Instances === undefined){
                        return false;
                    }

                    return true;
                }
            });
        };

        // TODO - clean up magic numbers
        $scope.calculateIndent = function(service){
            var indent = service.depth,
                offset = 1;

            if($scope.services.current && $scope.services.current.parent){
                offset = $scope.services.current.parent.depth + 2;
            }

            return $scope.indent(indent - offset);
        };


        function init(){
            $scope.name = "servicedetails";
            $scope.params = $routeParams;

            $scope.breadcrumbs = [
                { label: 'breadcrumb_deployed', url: '/apps' }
            ];

            $scope.vhostsTable = {
                sorting: {
                    Name: "asc"
                }
            };
            $scope.ipsTable = {
                sorting: {
                    ServiceName: "asc"
                }
            };
            $scope.configTable = {
                sorting: {
                    Filename: "asc"
                }
            };
            $scope.instancesTable = {
                sorting: {
                    "model.InstanceID": "asc"
                },
                // instead of watching for a change, always
                // reload at a specified interval
                watchExpression: (function(){
                    var last = new Date().getTime(),
                        now,
                        interval = 1000;

                    return function(){
                        now = new Date().getTime();
                        if(now - last > interval){
                            last = now;
                            return now;
                        }
                    };
                })()
            };
            $scope.scheduledTasksTable = {
                sorting: {
                    Schedule: "asc"
                }
            };

            // servicesTable should not be sortable since it
            // is a hierarchy.
            $scope.servicesTable = {};

            // setup initial state
            $scope.services = {
                data: servicesFactory.serviceTree,
                mapped: servicesFactory.serviceMap,
                current: servicesFactory.get($scope.params.serviceId)
            };

            $scope.ips = {};

            // if the current service changes, update
            // various service controller thingies
            $scope.$watch(function() {
                // if no current service is set, try to set one
                if(!$scope.services.current) {
                    $scope.services.current = servicesFactory.get($scope.params.serviceId);
                }

                if($scope.services.current) {
                    return $scope.services.current.isDirty();
                } else {
                    // there is no current service
                    console.warn("current service not yet available");
                    return undefined;
                }
            }, $scope.update);

            hostsFactory.activate();
            hostsFactory.update();

            servicesFactory.activate();
            servicesFactory.update();

            $scope.$on("$destroy", function() {
                servicesFactory.deactivate();
                hostsFactory.deactivate();
            });
        }

        // kick off controller
        init();



        function makeCrumbs(current){
            var crumbs = [{
                label: current.name,
                itemClass: "active",
                id: current.id
            }];

            (function recurse(service){
                if(service){
                    crumbs.unshift({
                        label: service.name,
                        url: "/services/"+ service.id,
                        id: service.id
                    });
                    recurse(service.parent);
                }
            })(current.parent);

            crumbs.unshift({
                label: "Applications",
                url: "/apps"
            });

            return crumbs;
        }
    }]);
})();
