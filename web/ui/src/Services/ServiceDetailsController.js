/* globals controlplane: true */

/* ServiceDetailsController
 * Displays details of a specific service
 */
(function () {
    'use strict';

    // share angular services outside of angular context
    var $notification, $q, resourcesFactory, utils;

    // service types
    var ISVC = "isvc",           // internal service
        APP = "app",             // service with no parent
        META = "meta",           // service with children but no startup command
        DEPLOYING = "deploying"; // service whose parent is still being deployed

    class Instance {

        constructor(instance) {
            this.name = instance.Name;
            this.model = Object.freeze(instance);
            this.id = buildStateId(this.model.HostID, this.model.ServiceID, this.model.InstanceID);

            this.resources = {
                RAMCommitment: utils.parseEngineeringNotation(instance.RAMCommitment)
            };

            this.updateState({
                Health: instance.HealthStatus,
                MemoryUsage: instance.MemoryUsage
            });
        }

        resourcesGood() {
            return this.resources.RAMLast < this.resources.RAMCommitment;
        }

        updateState(status) {
            this.resources.RAMAverage = Math.max(0, status.MemoryUsage.Avg);
            this.resources.RAMLast = Math.max(0, status.MemoryUsage.Cur);
            this.resources.RAMMax = Math.max(0, status.MemoryUsage.Max);

            // var hc = {};
            // for (var name in instance.HealthChecks) {
            //     hc[name] = instance.HealthChecks[name].Status;
            // }
            // this.healthChecks = hc;
        }

    }

    function buildStateId(hostid, serviceid, instanceid) {
        return `${hostid}-${serviceid}-${instanceid}`;
    }


    class Service {
        constructor(service) {
            // these properties are for convenience
            this.name = service.Name;
            this.id = service.ID;
            // NOTE: desiredState is mutable to improve UX
            this.desiredState = service.DesiredState;
            this.children = [];
            this.instances = [];

            // make service immutable
            this.model = Object.freeze(service);
            this.evaluateServiceType();
            this.touch();
        }

        evaluateServiceType() {
            // infer service type
            this.type = [];
            if (this.model.ID.indexOf("isvc-") !== -1) {
                this.type.push(ISVC);
            }

            if (!this.model.ParentServiceID) {
                this.type.push(APP);
            }

            if (this.children.length && !this.model.Startup) {
                this.type.push(META);
            }

            if (this.parent && this.parent.isDeploying()) {
                this.type.push(DEPLOYING);
            }
        }

        // fills out service object
        fetchAll() {
            this.fetchEndpoints();
            this.fetchAddresses();
            this.fetchConfigs();
            this.fetchServiceChildren();
        }

        fetchAddresses(force) {
            fetch.call(this, "getServiceIpAssignments", "addresses", force);
        }

        fetchConfigs(force) {
            fetch.call(this, "getServiceConfigs", "configs", force);
        }

        fetchEndpoints(force) {
            fetch.call(this, "getServicePublicEndpoints", "publicEndpoints", force);
        }

        fetchInstances() {
            let deferred = $q.defer();
            resourcesFactory.v2.getServiceInstances(this.id)
                .then(data => {
                    console.log(`fetched ${data.length} instances from getServiceInstances for id ${this.id}`);
                    this.instances = data.map(i => new Instance(i));
                    this.touch();
                    deferred.resolve();
                },
                error => {
                    console.warn(error);
                    deferred.reject();
                });

            return deferred.promise;
        }

        fetchServiceChildren(force) {
            // fetch.call(this, "getServiceChildren", "subservices", force);
            if (this.subservices) {
                return;
            }
            resourcesFactory.v2.getServiceChildren(this.id)
                .then(data => {
                    console.log(`fetched ${data.length} children services from getServiceChildren for id ${this.id}`);
                    this.subservices = data.map(s => new Service(s));
                    this.touch();
                },
                error => {
                    console.warn(error);
                });
        }

        // fast-moving state for this service and its instances
        // note: returns a promise that resolves with a single status object
        getStatus() {
            let deferred = $q.defer();

            resourcesFactory.v2.getServiceStatus(this.id)
                .then(results => {
                    if (results.length && !results[0].NotFound) {
                        deferred.resolve(results[0]);
                    } else {
                        deferred.reject(`Could not get service status for id ${this.id}`);
                    }
                }, error => {
                    deferred.reject(error);
                });
            return deferred.promise;
        }

        hasInstances() {
            return !!this.instances.length;
        }

        isIsvc() {
            return !!~this.type.indexOf(ISVC);
        }

        hasChildren() {
            return this.model.HasChildren;
        }

        stopInstance(instance) {
            resourcesFactory.killRunning(instance.model.HostID, instance.id)
                .success(() => {
                    this.touch();
                })
                .error((data, status) => {
                    $notification.create("Stop Instance failed", data.Detail).error();
                });
            
        }


        // mark services updated to trigger render via $watch
        touch() {
            this.lastUpdate = new Date().getTime();
        }

        // kicks off request to update instances and service state
        fetchAllStates() {
            $q.all([this.fetchInstances(), this.getStatus()])
                .then(results => {
                    this.updateState(results[1]);
                }, error => {
                    console.log("Unable to update instance states");
                    // TODO: Error
                });
        }

        // updates service state and instances states        
        updateState(status) {

            // update service status
            this.desiredState = status.DesiredState;

            let statusMap = {};
            status.Status.forEach(s => statusMap[s.InstanceID] = s);

            // update instance status
            this.instances.forEach(i => {
                let s = statusMap[i.id];
                // make sure status exists for instance
                if (!s) {
                    console.log(`Service instance ${i.id} has no status. Skipping status update.`);
                    return;
                }
                i.updateState(s);
            });

            this.touch();
        }

    }

    function fetch(methodName, propertyName, force) {
        let deferred = $q.defer();

        if (this[propertyName] && !force) {
            deferred.resolve();
            return deferred.promise;
        }
        // NOTE: V2 API only
        // TODO: make sure methodName exists
        // TODO: error callback
        resourcesFactory.v2[methodName](this.id)
            .then(data => {
                console.log(`fetched ${data.length} ${propertyName} from ${methodName} for id ${this.id}`);
                this[propertyName] = data;
                this.touch();
                deferred.resolve();
            },
            error => {
                deferred.reject(error);
                console.warn(error);
            });

        return deferred.promise;
    }


    controlplane.controller("ServiceDetailsController",
        ["$scope", "$q", "$routeParams", "$location", "resourcesFactory",
            "authService", "$modalService", "$translate", "$notification",
            "$timeout", "miscUtils", "hostsFactory",
            "poolsFactory", "CCUIState", "$cookies", "areUIReady", "LogSearch",
            function ($scope, _$q, $routeParams, $location, _resourcesFactory,
                authService, $modalService, $translate, _$notification,
                $timeout, _utils, hostsFactory,
                poolsFactory, CCUIState, $cookies, areUIReady, LogSearch) {

                // api access via angular context
                $notification = _$notification;
                $q = _$q;
                resourcesFactory = _resourcesFactory;
                utils = _utils;

                // Ensure logged in
                authService.checkLogin($scope);
                $scope.resourcesFactory = resourcesFactory;
                $scope.hostsFactory = hostsFactory;

                $scope.defaultHostAlias = $location.host();
                if (utils.needsHostAlias($location.host())) {
                    resourcesFactory.getHostAlias().success(function (data) {
                        $scope.defaultHostAlias = data.hostalias;
                    });
                }

                //add Public Endpoint data
                $scope.publicEndpoints = { add: {} };

                //add service endpoint data
                $scope.exportedServiceEndpoints = {};

                $scope.click_pool = function (id) {
                    resourcesFactory.routeToPool(id);
                };

                $scope.click_host = function (id) {
                    resourcesFactory.routeToHost(id);
                };


                $scope.modalAddPublicEndpoint = function () {
                    areUIReady.lock();
                    $scope.protocols = [];
                    $scope.protocols.push({ Label: "HTTPS", UseTLS: true, Protocol: "https" });
                    $scope.protocols.push({ Label: "HTTP", UseTLS: false, Protocol: "http" });
                    $scope.protocols.push({ Label: "Other, secure (TLS)", UseTLS: true, Protocol: "" });
                    $scope.protocols.push({ Label: "Other, non-secure", UseTLS: false, Protocol: "" });

                    // default public endpoint options
                    $scope.publicEndpoints.add = {
                        type: "port",
                        app_ep: $scope.exportedServiceEndpoints.data[0],
                        name: "",
                        host: $scope.defaultHostAlias,
                        port: "",
                        protocol: $scope.protocols[0],
                    };

                    // returns an error string if newPublicEndpoint's vhost is invalid
                    var validateVHost = function (newPublicEndpoint) {
                        var name = newPublicEndpoint.name;

                        // if no port
                        if (!name || !name.length) {
                            return "Missing Name";
                        }

                        // if name already exists
                        for (var i in $scope.publicEndpoints.data) {
                            if (name === $scope.publicEndpoints.data[i].Name) {
                                return "Name already exists: " + newPublicEndpoint.name;
                            }
                        }

                        // if invalid characters
                        var re = /^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$/;
                        if (!re.test(name)) {
                            return $translate.instant("vhost_name_invalid") + " " + newPublicEndpoint.name;
                        }
                    };

                    // returns an error string if newPublicEndpoint's port is invalid
                    var validatePort = function (newPublicEndpoint) {
                        var host = newPublicEndpoint.host;
                        var port = newPublicEndpoint.port;

                        if (!host || !host.length) {
                            return "Missing host name";
                        }

                        // if invalid characters
                        var re = /^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$/;
                        if (!re.test(host)) {
                            return $translate.instant("host_name_invalid") + ": " + host;
                        }

                        // if no port
                        if (!port || !port.length) {
                            return "Missing port";
                        }

                        if (+port < 1 || +port > 65536) {
                            return "Port must be between 1 and 65536";
                        }
                    };

                    $modalService.create({
                        templateUrl: "add-public-endpoint.html",
                        model: $scope,
                        title: "add_public_endpoint",
                        actions: [
                            {
                                role: "cancel",
                                action: function () {
                                    this.close();
                                }
                            }, {
                                role: "ok",
                                label: "add_public_endpoint_confirm",
                                action: function () {
                                    var newPublicEndpoint = $scope.publicEndpoints.add;

                                    if (this.validate(newPublicEndpoint)) {
                                        // disable ok button, and store the re-enable function
                                        var enableSubmit = this.disableSubmitButton();

                                        $scope.addPublicEndpoint(newPublicEndpoint)
                                            .success(function (data, status) {
                                                $notification.create("Added public endpoint").success();
                                                this.close();
                                            }.bind(this))
                                            .error(function (data, status) {
                                                this.createNotification("Unable to add public endpoint", data.Detail).error();
                                                enableSubmit();
                                            }.bind(this));

                                    }
                                }
                            }
                        ],

                        validate: function (newPublicEndpoint) {
                            // if no service endpoint selected
                            if (!newPublicEndpoint.app_ep) {
                                this.createNotification("Unable to add Public Endpoint", "No service endpoint selected").error();
                                return false;
                            }


                            // perform type specific validation
                            if (newPublicEndpoint.type === "vhost") {
                                var err = validateVHost(newPublicEndpoint);
                                if (err) {
                                    this.createNotification("Unable to add Public Endpoint", err).error();
                                } else {
                                    return true;
                                }
                            } else if (newPublicEndpoint.type === "port") {
                                var err = validatePort(newPublicEndpoint);
                                if (err) {
                                    this.createNotification("Unable to add Public Endpoint", err).error();
                                    return false;
                                } else {
                                    return true;
                                }
                            }
                        },
                        onShow: () => {
                            areUIReady.unlock();
                        }
                    });
                };


                $scope.addPublicEndpoint = function (newPublicEndpoint) {
                    var serviceId = newPublicEndpoint.app_ep.ApplicationId;
                    var serviceName = newPublicEndpoint.app_ep.Application;
                    var serviceEndpoint = newPublicEndpoint.app_ep.ServiceEndpoint;
                    if (newPublicEndpoint.type === "vhost") {
                        var vhostName = newPublicEndpoint.name;
                        return resourcesFactory.addVHost(serviceId, serviceName, serviceEndpoint, vhostName);
                    } else if (newPublicEndpoint.type === "port") {
                        var port = newPublicEndpoint.host + ":" + newPublicEndpoint.port;
                        var usetls = newPublicEndpoint.protocol.UseTLS;
                        var protocol = newPublicEndpoint.protocol.Protocol;
                        return resourcesFactory.addPort(serviceId, serviceName, serviceEndpoint, port, usetls, protocol);
                    }
                };

                // modalAssignIP opens a modal view to assign an ip address to a service
                $scope.modalAssignIP = function (ip, poolID) {
                    $scope.ips.assign = { 'ip': ip, 'value': null };
                    resourcesFactory.getPoolIPs(poolID)
                        .success(function (data) {
                            var options = [{ 'Value': 'Automatic', 'IPAddr': '' }];

                            var i, IPAddr, value;
                            //host ips
                            if (data && data.HostIPs) {
                                for (i = 0; i < data.HostIPs.length; ++i) {
                                    IPAddr = data.HostIPs[i].IPAddress;
                                    value = 'Host: ' + IPAddr + ' - ' + data.HostIPs[i].InterfaceName;
                                    options.push({ 'Value': value, 'IPAddr': IPAddr });
                                    // set the default value to the currently assigned value
                                    if ($scope.ips.assign.ip.IPAddr === IPAddr) {
                                        $scope.ips.assign.value = options[options.length - 1];
                                    }
                                }
                            }

                            //virtual ips
                            if (data && data.VirtualIPs) {
                                for (i = 0; i < data.VirtualIPs.length; ++i) {
                                    IPAddr = data.VirtualIPs[i].IP;
                                    value = "Virtual IP: " + IPAddr;
                                    options.push({ 'Value': value, 'IPAddr': IPAddr });
                                    // set the default value to the currently assigned value
                                    if ($scope.ips.assign.ip.IPAddr === IPAddr) {
                                        $scope.ips.assign.value = options[options.length - 1];
                                    }
                                }
                            }

                            //default to automatic
                            if (!$scope.ips.assign.value) {
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
                                    }, {
                                        role: "ok",
                                        label: "assign_ip",
                                        action: function () {
                                            if (this.validate()) {
                                                // disable ok button, and store the re-enable function
                                                var enableSubmit = this.disableSubmitButton();

                                                $scope.assignIP()
                                                    .success(function (data, status) {
                                                        $notification.create("Added IP", data.Detail).success();
                                                        this.close();
                                                    }.bind(this))
                                                    .error(function (data, status) {
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

                $scope.anyServicesExported = function (service) {
                    // if(service){
                    //     for (var i in service.Endpoints) {
                    //         if (service.Endpoints[i].Purpose === "export") {
                    //             return true;
                    //         }
                    //     }
                    //     for (var j in service.children) {
                    //         if ($scope.anyServicesExported(service.children[j])) {
                    //             return true;
                    //         }
                    //     }
                    // }
                    // return false;
                    return true;
                };


                $scope.assignIP = function () {
                    var serviceID = $scope.ips.assign.ip.ServiceID;
                    var IP = $scope.ips.assign.value.IPAddr;
                    return resourcesFactory.assignIP(serviceID, IP)
                        .success(function (data, status) {
                            // HACK: update(true) forces a full update to
                            // work around issue https://jira.zenoss.com/browse/CC-939
                            // servicesFactory.update(true);
                        });
                };


                $scope.publicEndpointProtocol = function (publicEndpoint) {
                    if (publicEndpoint.type === "vhost") {
                        return "https";
                    } else {
                        if (publicEndpoint.Protocol !== "") {
                            return publicEndpoint.Protocol;
                        }
                        if (publicEndpoint.UseTLS) {
                            return "other (TLS)";
                        }
                        return "other";
                    }
                };

                $scope.indent = function (depth) {
                    return { 'padding-left': (15 * depth) + "px" };
                };

                // sets a service to start, stop or restart state
                $scope.setServiceState = function (service, state, skipChildren) {
                    service[state](skipChildren).error(function (data, status) {
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
                $scope.clickRunning = function (service, state) {
                    var filterFn = serviceStateChangeFilters[state];
                    var childCount = utils.countTheKids(service, filterFn);

                    // if the service has affected children, check if the user
                    // wants to start just the service, or the service and children
                    if (childCount > 0) {
                        $scope.modal_confirmSetServiceState(service, state, childCount);

                        // if no children, just start the service
                    } else {
                        $scope.setServiceState(service, state);
                    }
                    // servicesFactory.updateHealth();
                };

                // verifies if use wants to start parent service, or parent
                // and all children
                $scope.modal_confirmSetServiceState = function (service, state, childCount) {
                    $modalService.create({
                        template: ["<h4>" + $translate.instant("choose_services_" + state) + "</h4><ul>",
                            "<li>" + $translate.instant(state + "_service_name", { name: "<strong>" + service.name + "</strong>" }) + "</li>",
                            "<li>" + $translate.instant(state + "_service_name_and_children", { name: "<strong>" + service.name + "</strong>", count: "<strong>" + childCount + "</strong>" }) + "</li></ul>"
                        ].join(""),
                        model: $scope,
                        title: $translate.instant(state + "_service"),
                        actions: [
                            {
                                role: "cancel"
                            }, {
                                role: "ok",
                                classes: " ",
                                label: $translate.instant(state + "_service"),
                                action: function () {
                                    // the arg here explicitly prevents child services
                                    // from being started
                                    $scope.setServiceState(service, state, true);
                                    this.close();
                                }
                            }, {
                                role: "ok",
                                label: $translate.instant(state + "_service_and_children", { count: childCount }),
                                action: function () {
                                    $scope.setServiceState(service, state);
                                    this.close();
                                }
                            }
                        ]
                    });
                };


                $scope.clickEndpointEnable = function (publicEndpoint) {
                    if (publicEndpoint.type === "vhost") {
                        resourcesFactory.enableVHost(publicEndpoint.ApplicationId, publicEndpoint.Application, publicEndpoint.ServiceEndpoint, publicEndpoint.Name)
                            .error((data, status) => {
                                $notification.create("Enable Public Endpoint failed", data.Detail).error();
                            });
                    } else if (publicEndpoint.type === "port") {
                        resourcesFactory.enablePort(publicEndpoint.ApplicationId, publicEndpoint.Application, publicEndpoint.ServiceEndpoint, publicEndpoint.PortAddr)
                            .error((data, status) => {
                                $notification.create("Enable Public Endpoint failed", data.Detail).error();
                            });
                    }
                };


                $scope.clickEndpointDisable = function (publicEndpoint) {
                    if (publicEndpoint.type === "vhost") {
                        resourcesFactory.disableVHost(publicEndpoint.ApplicationId, publicEndpoint.Application, publicEndpoint.ServiceEndpoint, publicEndpoint.Name)
                            .error((data, status) => {
                                $notification.create("Disable Public Endpoint failed", data.Detail).error();
                            });
                    } else if (publicEndpoint.type === "port") {
                        resourcesFactory.disablePort(publicEndpoint.ApplicationId, publicEndpoint.Application, publicEndpoint.ServiceEndpoint, publicEndpoint.PortAddr)
                            .error((data, status) => {
                                $notification.create("Disable Public Endpoint failed", data.Detail).error();
                            });
                    }
                };

                $scope.clickEditContext = function () {
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
                            }, {
                                role: "ok",
                                label: $translate.instant("btn_save_changes"),
                                action: function () {
                                    // disable ok button, and store the re-enable function
                                    var enableSubmit = this.disableSubmitButton();

                                    $scope.editableService.Context = makeStorableContext($scope.editableContext);

                                    $scope.updateService($scope.editableService)
                                        .success(function (data, status) {
                                            $notification.create("Updated service", $scope.editableService.ID).success();
                                            this.close();
                                        }.bind(this))
                                        .error(function (data, status) {
                                            this.createNotification("Update service failed", data.Detail).error();
                                            enableSubmit();
                                        }.bind(this));
                                }
                            }
                        ],
                        onShow: function () {
                            $scope.codemirrorRefresh = true;
                        },
                        onHide: function () {
                            $scope.codemirrorRefresh = false;
                        }
                    });
                };

                function makeEditableContext(context) {
                    var editableContext = "";
                    for (var key in context) {
                        editableContext += key + " " + context[key] + "\n";
                    }
                    if (!editableContext) { editableContext = ""; }
                    return editableContext;
                }
                function makeStorableContext(context) {
                    //turn editableContext into a JSON object
                    var lines = context.split("\n"),
                        storable = {};

                    lines.forEach(function (line) {
                        var delimitIndex, key, val;

                        if (line !== "") {
                            delimitIndex = line.indexOf(" ");
                            if (delimitIndex !== -1) {
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


                $scope.clickRemovePublicEndpoint = function (publicEndpoint) {

                    $modalService.create({
                        template: $translate.instant("remove_public_endpoint") + ": <strong>" +
                        (publicEndpoint.Name ? publicEndpoint.Name : "port " + publicEndpoint.PortAddr) + "</strong><br><br>",
                        model: $scope,
                        title: "remove_public_endpoint",
                        actions: [
                            {
                                role: "cancel"
                            }, {
                                role: "ok",
                                label: "remove_public_endpoint_confirm",
                                classes: "btn-danger",
                                action: function () {
                                    if (publicEndpoint.type === "vhost") {
                                        resourcesFactory.removeVHost(publicEndpoint.ApplicationId, publicEndpoint.ServiceEndpoint, publicEndpoint.Name)
                                            .success(() => {
                                                // servicesFactory.update();
                                                $notification.create("Removed Public Endpoint", publicEndpoint.Name).success();
                                            })
                                            .error((data, status) => {
                                                $notification.create("Remove Public Endpoint failed", data.Detail).error();
                                            });
                                    } else if (publicEndpoint.type === "port") {
                                        resourcesFactory.removePort(publicEndpoint.ApplicationId, publicEndpoint.ServiceEndpoint, publicEndpoint.PortAddr)
                                            .success(() => {
                                                // servicesFactory.update();
                                                $notification.create("Removed Public Endpoint", publicEndpoint.PortName).success();
                                            })
                                            .error((data, status) => {
                                                $notification.create("Remove Public Endpoint failed", data.Detail).error();
                                            });
                                    }
                                    this.close();
                                }
                            }
                        ]
                    });
                };

                $scope.editConfig = function (config) {
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
                        title: $translate.instant("title_edit_config") + " - " + $scope.selectedConfig,
                        bigModal: true,
                        actions: [
                            {
                                role: "cancel"
                            }, {
                                role: "ok",
                                label: "save",
                                action: function () {
                                    if (this.validate()) {
                                        // disable ok button, and store the re-enable function
                                        var enableSubmit = this.disableSubmitButton();

                                        $scope.updateService($scope.editableService)
                                            .success(function (data, status) {
                                                $notification.create("Updated service", $scope.editableService.ID).success();
                                                this.close();
                                            }.bind(this))
                                            .error(function (data, status) {
                                                this.createNotification("Update service failed", data.Detail).error();
                                                enableSubmit();
                                            }.bind(this));
                                    }
                                }
                            }
                        ],
                        validate: function () {
                            // TODO - actually validate
                            return true;
                        },
                        onShow: function () {
                            $scope.codemirrorRefresh = true;
                        },
                        onHide: function () {
                            $scope.codemirrorRefresh = false;
                        }
                    });
                };

                $scope.viewLog = function (instance) {
                    let logScope = $scope.$new(true);

                    resourcesFactory.getInstanceLogs(instance.model.ServiceID, instance.id)
                        .success(function (log) {
                            logScope.log = log.Detail;
                            $modalService.create({
                                templateUrl: "view-log.html",
                                model: logScope,
                                title: "title_log",
                                bigModal: true,
                                actions: [
                                    {
                                        role: "cancel",
                                        label: "close"
                                    }, {
                                        classes: "btn-primary",
                                        label: "refresh",
                                        icon: "glyphicon-repeat",
                                        action: function () {
                                            var textarea = this.$el.find("textarea");
                                            resourcesFactory.getInstanceLogs(instance.model.ServiceID, instance.id)
                                                .success(function (log) {
                                                    logScope.log = log.Detail;
                                                    textarea.scrollTop(textarea[0].scrollHeight - textarea.height());
                                                })
                                                .error((data, status) => {
                                                    this.createNotification("Unable to fetch logs", data.Detail).error();
                                                });
                                        }
                                    }, {
                                        classes: "btn-primary",
                                        label: "download",
                                        action: function () {
                                            utils.downloadFile('/services/' + instance.model.ServiceID + '/' + instance.id + '/logs/download');
                                        },
                                        icon: "glyphicon-download"
                                    }
                                ],
                                onShow: function () {
                                    var textarea = this.$el.find("textarea");
                                    textarea.scrollTop(textarea[0].scrollHeight - textarea.height());
                                }
                            });
                        })
                        .error(function (data, status) {
                            $notification.create("Unable to fetch logs", data.Detail).error();
                        });
                };

                $scope.validateService = function () {
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

                $scope.updateService = function (newService) {
                    if ($scope.validateService()) {
                        return resourcesFactory.updateService($scope.services.current.model.ID, newService)
                            .success((data, status) => {
                                // servicesFactory.update();
                                this.editableService = {};
                            });
                    }
                };

                $scope.subNavClick = function (crumb) {
                    if (crumb.id) {
                        $scope.routeToService(crumb.id);
                    } else {
                        // TODO - just call subnavs usual function
                        $location.path(crumb.url);
                    }
                };

                // grab default kibana search configs and adjust
                // the query to look for this specific service
                $scope.getServiceLogURL = function (service) {
                    if (!service) {
                        return "";
                    }
                    let appConfig = LogSearch.getDefaultAppConfig(),
                        globalConfig = LogSearch.getDefaultGlobalConfig();

                    appConfig.query = {
                        query_string: {
                            analyze_wildcard: true,
                            query: `fields.service:${service.id} AND fields.instance:* AND message:*`
                        }
                    };
                    appConfig.columns = ["fields.instance", "message"];

                    return LogSearch.getURL(appConfig, globalConfig);
                };

                // grab default kibana search configs and adjust
                // the query to look for this specific service instance
                $scope.getInstanceLogURL = function (instance) {
                    let appConfig = LogSearch.getDefaultAppConfig(),
                        globalConfig = LogSearch.getDefaultGlobalConfig();

                    appConfig.query = {
                        query_string: {
                            analyze_wildcard: true,
                            query: `fields.service:${instance.model.ServiceID} AND fields.instance:${instance.model.InstanceID} AND message:*`
                        }
                    };
                    appConfig.columns = ["message"];

                    return LogSearch.getURL(appConfig, globalConfig);

                };

                $scope.routeToService = function (id, e) {
                    // if an event is present, we may
                    // need to prevent it from performing
                    // default navigation behavior
                    if (e) {
                        // ctrl click opens in new tab,
                        // so allow that to happen and don't
                        // bother routing the current view
                        if (e.ctrlKey) {
                            return;
                        }

                        // if middle click, don't update
                        // current view
                        if (e.button === 1) {
                            return;
                        }

                        // otherwise, prevent default so
                        // we can handle the view routing
                        e.preventDefault();
                    }

                    $location.update_path("/services/" + id, true);
                    $scope.params.serviceId = id;
                    // $scope.services.current = servicesFactory.get($scope.params.serviceId);
                    $scope.update();
                };

                // restart all running instances for this service
                $scope.killRunningInstances = function (app) {
                    resourcesFactory.restartService(app.ID)
                        .error((data, status) => {
                            $notification.create("Stop Service failed", data.Detail).error();
                        });
                };

                $scope.startTerminal = function (app) {
                    window.open("http://" + window.location.hostname + ":50000");
                };



                $scope.getHostName = function (id) {
                    if (hostsFactory.get(id)) {
                        return hostsFactory.get(id).name;
                    } else {
                        // TODO - if unknown host, dont make linkable
                        // and use custom css to show unknown
                        return "unknown";
                    }
                };

                // expand/collapse state of service tree nodes
                $scope.serviceTreeState = CCUIState.get($cookies.get("ZUsername"), "serviceTreeState");
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

                $scope.toggleChildren = function (service) {
                    if (!$scope.services.current) {
                        console.warn("Cannot store toggle state: no current service");
                        return;
                    }

                    // stored state for the current service's
                    // service tree
                    var treeState = $scope.services.currentTreeState;

                    // if this service is marked as collapsed in
                    // this particular tree view, show its children
                    if (treeState[service].collapsed) {
                        treeState[service].collapsed = false;
                        console.log("expand  " + service);
                        // TODO: create Service object for [service]
                        // TODO: call fetchServiceChildren() on it.

                        // otherwise, hide its children
                    } else {
                        treeState[service].collapsed = true;
                        console.log("retract " + service);
                        $scope.hideChildren(service);
                    }
                };

                $scope.getService = function (id) {
                    let deferred = $q.defer();
                    $scope.resourcesFactory.v2.getService(id)
                        .then(function (data) {
                            console.log("got service id=" + id);
                            deferred.resolve(data);
                        },
                        function (error) {
                            console.warn(error);
                            deferred.reject(error);
                        });
                    return deferred.promise;
                };

                // $scope.getServiceChildren = function(id) {
                //     $scope.resourcesFactory.v2.getServiceChildren(id)
                //         .success(function(data){
                //             console.log(data.length + " children returned.");
                //             $scope.services.current.children = data;
                //             // $scope.insertTreeChildren(data.results);
                //         });
                // };

                $scope.getServiceEndpoints = function (id) {
                    let deferred = $q.defer();
                    $scope.resourcesFactory.v2.getServiceEndpoints(id)
                        .then(function (response) {
                            console.log("got service endpoints for id=" + id);
                            deferred.resolve(response.data);
                        },
                        function (response) {
                            console.warn(response.status + " " + response.statusText);
                            deferred.reject(response.statusText);
                        });
                    return deferred.promise;
                };

                $scope.getServices = function () {
                    $scope.resourcesFactory.v2.getServices()
                        .success(function (data) {
                            console.log(data.length + " top level children.");
                            // $scope.services.current = data.results;
                        });
                };

                $scope.insertTreeChildren = function (subservices) {
                    var par = subservices[0].ParentServiceID;
                    var index = $scope.services.data.findIndex(svc => svc.id === par);







                };

                $scope.hideChildren = function (service) {
                    // get the state of the current service's tree
                    var treeState = $scope.services.currentTreeState;

                    if (service.children) {
                        service.children.forEach(function (child) {
                            treeState[child.id].hidden = true;
                            $scope.hideChildren(child);
                        });
                    }
                };

                $scope.showChildren = function (service) {
                    var treeState = $scope.services.currentTreeState;

                    if (service.children) {
                        service.children.forEach(function (child) {
                            treeState[child.id].hidden = false;

                            // if this child service is not marked
                            // as collapsed, show its children
                            if (!treeState[child.id].collapsed) {
                                $scope.showChildren(child);
                            }
                        });
                    }
                };

                //we need to bring this function into scope so we can use ng-hide if an object is empty
                $scope.isEmptyObject = function (obj) {
                    return angular.equals({}, obj);
                };

                $scope.isIsvc = function (service) {
                    return service.isIsvc();
                };

                $scope.hasCurrentInstances = function () {
                    return $scope.services && $scope.services.current && $scope.services.current.hasInstances();
                };

                $scope.editCurrentService = function () {

                    // clone service for editing
                    $scope.editableService = angular.copy($scope.services.current.model);

                    $modalService.create({
                        templateUrl: "edit-service.html",
                        model: $scope,
                        title: "title_edit_service",
                        actions: [
                            {
                                role: "cancel"
                            }, {
                                role: "ok",
                                label: "btn_save_changes",
                                action: function () {
                                    if (this.validate()) {

                                        // disable ok button, and store the re-enable function
                                        var enableSubmit = this.disableSubmitButton();

                                        // update service with recently edited service
                                        $scope.updateService($scope.editableService)
                                            .success(function (data, status) {
                                                $notification.create("Updated service", $scope.editableService.ID).success();
                                                this.close();
                                            }.bind(this))
                                            .error(function (data, status) {
                                                this.createNotification("Update service failed", data.Detail).error();
                                                enableSubmit();
                                            }.bind(this));
                                    }
                                }
                            }
                        ],
                        validate: function () {
                            if ($scope.editableService.InstanceLimits.Min > $scope.editableService.Instances || $scope.editableService.Instances === undefined) {
                                return false;
                            }

                            return true;
                        }
                    });
                };

                // TODO - clean up magic numbers
                $scope.calculateIndent = function (service) {
                    var indent = service.depth,
                        offset = 1;

                    if ($scope.services.current && $scope.services.current.parent) {
                        offset = $scope.services.current.parent.depth + 2;
                    }

                    return $scope.indent(indent - offset);
                };

                $scope.update = function () {

                    console.log("UPDATE --------------");

                    if ($scope.services.current) {

                        $scope.services.current.fetchAll();
                        // setup breadcrumbs
                        $scope.breadcrumbs = makeCrumbs($scope.services.current);

                        // update serviceTreeState
                        $scope.serviceTreeState = CCUIState.get($cookies.get("ZUsername"), "serviceTreeState");

                        // update pools
                        $scope.pools = poolsFactory.poolList;

                        // create an entry in tree state for the
                        // current service
                        if (!($scope.services.current.id in $scope.serviceTreeState)) {
                            $scope.serviceTreeState[$scope.services.current.id] = {};
                        }
                        var treeState = $scope.serviceTreeState[$scope.services.current.id];

                        // initialize services as collapsed
                        if ($scope.services.current.subservices) {
                            $scope.services.current.subservices.forEach(svc => {
                                if (!treeState[svc.ID]) {
                                    console.log(`setting serviceTreeState[${svc.ID}] to collapsed`);
                                    treeState[svc.ID] = {
                                        hidden: false,
                                        collapsed: true
                                    };
                                }
                            });
                        }

                        // property for view to bind for tree state
                        $scope.services.currentTreeState = $scope.serviceTreeState[$scope.services.current.id];

                    }

                };


                function init() {
                    $scope.name = "servicedetails";
                    $scope.params = $routeParams;

                    $scope.breadcrumbs = [
                        { label: 'breadcrumb_deployed', url: '/apps' }
                    ];

                    $scope.publicEndpointsTable = {
                        sorting: {
                            ServiceEndpoint: "asc"
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
                        watchExpression: (function () {
                            var last = new Date().getTime(),
                                now,
                                interval = 1000;

                            return function () {
                                now = new Date().getTime();
                                if (now - last > interval) {
                                    last = now;
                                    return now;
                                }
                            };
                        })()
                    };

                    // servicesTable should not be sortable since it
                    // is a hierarchy.
                    $scope.servicesTable = {
                        disablePagination: true
                    };

                    // setup initial state
                    $scope.services = {
                        // data: servicesFactory.serviceTree,
                        // mapped: servicesFactory.serviceMap,
                        // current: servicesFactory.get($scope.params.serviceId)
                    };

                    // $scope.getServices();

                    $scope.ips = {};
                    $scope.pools = [];

                    // if the current service changes, update
                    // various service controller thingies


                    // $scope.$watch(function() {
                    //     // if no current service is set, try to set one
                    //     if(!$scope.services.current) {
                    //         // v2 call with id = $scope.params.serviceId
                    //         $scope.getService($scope.params.serviceId)
                    //             .then( function(model){
                    //                 $scope.services.current = new Service(model);
                    //             });
                    //         // $scope.services.current = servicesFactory.get($scope.params.serviceId);
                    //     }

                    //     if($scope.services.current) {
                    //         return $scope.services.current.isDirty();
                    //     } else {
                    //         // there is no current service
                    //         console.warn("current service not yet available");
                    //         return undefined;
                    //     }
                    // }, $scope.update);


                    $scope.getService($scope.params.serviceId)
                        .then(function (model) {
                            $scope.services.current = new Service(model);
                        });

                    $scope.$watch("services.current.lastUpdate", $scope.update);

                    hostsFactory.activate();
                    hostsFactory.update();

                    setInterval(function () {
                        if ($scope.services.current) {
                            $scope.services.current.fetchAllStates();
                        }
                    }, 5000);

                    // servicesFactory.activate();
                    // servicesFactory.update();

                    poolsFactory.activate();
                    poolsFactory.update();

                    $scope.$on("$destroy", function () {
                        // servicesFactory.deactivate();
                        hostsFactory.deactivate();
                        poolsFactory.deactivate();
                    });



                }

                // kick off controller
                init();



                function makeCrumbs(current) {
                    var crumbs = [{
                        label: current.name,
                        itemClass: "active",
                        id: current.id
                    }];

                    (function recurse(service) {
                        if (service) {
                            crumbs.unshift({
                                label: service.name,
                                url: "/services/" + service.id,
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
