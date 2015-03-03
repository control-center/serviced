/* globals controlplane: true */

/* ServiceDetailsController
 * Displays details of a specific service
 */
(function() {
    'use strict';

    controlplane.controller("ServiceDetailsController", ["$scope", "$q", "$routeParams", "$location", "resourcesFactory", "authService", "$modalService", "$translate", "$notification", "$timeout", "servicesFactory", "miscUtils", "hostsFactory",
    function($scope, $q, $routeParams, $location, resourcesFactory, authService, $modalService, $translate, $notification, $timeout, servicesFactory, utils, hostsFactory){
        // Ensure logged in
        authService.checkLogin($scope);
        $scope.name = "servicedetails";
        $scope.params = $routeParams;
        $scope.resourcesFactory = resourcesFactory;
        $scope.hostsFactory = hostsFactory;

        $scope.defaultHostAlias = $location.host();
        if(utils.needsHostAlias($location.host())){
            resourcesFactory.getHostAlias().success(function(data) {
                $scope.defaultHostAlias = data.hostalias;
            });
        }

        $scope.breadcrumbs = [
            { label: 'breadcrumb_deployed', url: '#/apps' }
        ];

        $scope.vhosts = utils.buildTable('Name', [
            { id: 'Name', name: 'vhost_name'},
            { id: 'Application', name: 'vhost_application'},
            { id: 'ServiceEndpoint', name: 'vhost_service_endpoint'},
            { id: 'Name', name: 'vhost_url'}
        ]);

        $scope.ips = utils.buildTable('ServiceName', [
            { id: 'ServiceName', name: 'tbl_virtual_ip_service'},
            { id: 'AssignmentType', name: 'tbl_virtual_ip_assignment_type'},
            { id: 'HostName', name: 'tbl_virtual_ip_host'},
            { id: 'PoolID', name: 'tbl_virtual_ip_pool'},
            { id: 'IPAddr', name: 'tbl_virtual_ip'}
        ]);

        //add vhost data (includes name, app & service endpoint)
        $scope.vhosts.add = {};

        $scope.click_app = function(id) {
            $location.path('/services/' + id);
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

                //host ips
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
                    servicesFactory.update();
                });
        };

        $scope.vhost_url = function(vhost) {
            var port = location.port === "" ? "" : ":"+location.port;
            var host = vhost.indexOf('.') === -1 ? vhost + "." + $scope.defaultHostAlias : vhost;
            return location.protocol + "//" + host + port;
        };

        $scope.indent = function(depth){
            return {'padding-left': (20*depth) + "px"};
        };

        $scope.clickRunning = function(app, status){
            app[status]();
            servicesFactory.updateHealth();
        };

        $scope.clickRunningApp = function(app, status) {

            // if this service has children and startup command, ask the user
            // if we should start service + children, or just service
            if(app.children && app.children.length && app.model.Startup){
                var children = app.children || [],
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
                        "<li>"+ $translate.instant(status +"_service_name", {name: "<strong>"+app.name+"</strong>"}) +"</li>",
                        "<li>"+ $translate.instant(status +"_service_name_and_children", {name: "<strong>"+app.name+"</strong>", count: "<strong>"+childCount+"</strong>"}) +"</li></ul>"
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
                                // the arg here explicitly prevents child services
                                // from being started
                                app[status](true);
                                this.close();
                            }
                        },{
                            role: "ok",
                            label: $translate.instant(status +"_service_and_children", {count: childCount}),
                            action: function(){
                                app[status]();
                                this.close();
                            }
                        }
                    ]
                });

            // this service has no children or no startup command,
            // so start it the usual way
            } else {
                $scope.clickRunning(app, status);
            }

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
                .error((data, status) => {
                    this.createNotification("Unable to fetch logs", data.Detail).error();
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

        $scope.update = function(){
            if($scope.services.current){
                $scope.services.subservices = $scope.services.current.descendents;
                $scope.vhosts.data = $scope.services.current.hosts;
                $scope.ips.data = $scope.services.current.addresses;
                
                // update instances
                $scope.services.current.getServiceInstances();

                // setup breadcrumbs
                $scope.breadcrumbs = makeCrumbs($scope.services.current);
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

        $scope.toggleChildren = function($event, app){
            var $e = $($event.target);
            if($e.is(".glyphicon-chevron-down")){
                hideChildren(app);
            } else {
                showChildren(app);
            }
        };

        //we need to bring this function into scope so we can use ng-hide if an object is empty
        $scope.isEmptyObject = function(obj){
            return angular.equals({}, obj);
        };

        $scope.isIsvc = function(service){
            return service.type === "isvc";
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
            // setup initial state
            $scope.services = {
                data: servicesFactory.serviceTree,
                mapped: servicesFactory.serviceMap,
                current: servicesFactory.get($scope.params.serviceId)
            };

            // if the current service changes, update
            // various service controller thingies
            $scope.$watch(function(){
                // if no current service is set, try to set one
                if(!$scope.services.current){
                    $scope.services.current = servicesFactory.get($scope.params.serviceId);
                }

                if($scope.services.current){
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

            $scope.$on("$destroy", function(){
                servicesFactory.deactivate();
                hostsFactory.deactivate();
            });

        }

        // kick off controller
        init();

        function hideChildren(app){
            app.children.forEach(function(child){
                $("tr[data-id='" + child.id + "'] td").hide();
                hideChildren(child);
            });

            //update icons
            var $e = $("tr[data-id='"+app.id+"'] td .glyphicon-chevron-down");
            $e.removeClass("glyphicon-chevron-down");
            $e.addClass("glyphicon-chevron-right");
        }

        function showChildren(app){
            app.children.forEach(function(child){
                $("tr[data-id='" + child.id + "'] td").show();
                showChildren(child);
            });

            //update icons
            var $e = $("tr[data-id='"+app.id+"'] td .glyphicon-chevron-right");
            $e.removeClass("glyphicon-chevron-right");
            $e.addClass("glyphicon-chevron-down");
        }

        function makeCrumbs(current){
            var crumbs = [{
                label: current.name,
                itemClass: "active"
            }];

            (function recurse(service){
                if(service){
                    crumbs.unshift({
                        label: service.name,
                        url: "#/services/"+ service.id
                    });
                    recurse(service.parent);
                }
            })(current.parent);

            crumbs.unshift({
                label: "Applications",
                url: "#/apps"
            });

            return crumbs;
        }
    }]);
})();
