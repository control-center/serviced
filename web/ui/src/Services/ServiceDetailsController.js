/* globals controlplane: true */

/* ServiceDetailsController
 * Displays details of a specific service
 */
(function () {
    'use strict';

    // share angular services outside of angular context
    let $notification, $q, resourcesFactory, utils;

    controlplane.controller("ServiceDetailsController", [
            "$scope", "$q", "$routeParams", "$location", "resourcesFactory",
            "authService", "$modalService", "$translate", "$notification",
            "$timeout", "miscUtils", "Service",
            "CCUIState", "$cookies", "areUIReady", "LogSearch",
            "$filter", "Pool",
            function ($scope, _$q, $routeParams, $location, _resourcesFactory,
                authService, $modalService, $translate, _$notification,
                $timeout, _utils, Service,
                CCUIState, $cookies, areUIReady, LogSearch,
                $filter, Pool) {

                // api access via angular context
                $notification = _$notification;
                $q = _$q;
                resourcesFactory = _resourcesFactory;
                utils = _utils;

                // Ensure logged in
                authService.checkLogin($scope);

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

                $scope.clickPool = function (id) {
                    resourcesFactory.routeToPool(id);
                };

                $scope.clickHost = function (id) {
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
                        endpoint: $scope.currentService.exportedServiceEndpoints[0],
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
                            return $translate.instant("vhost_name_invalid") + " " + utils.escapeHTML(newPublicEndpoint.name);
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
                            return $translate.instant("host_name_invalid") + ": " + utils.escapeHTML(host);
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
                                                // reload the table
                                                refreshEndpoints();
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
                            if (!newPublicEndpoint.endpoint) {
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
                    var serviceId = newPublicEndpoint.endpoint.ServiceID;
                    var serviceName = newPublicEndpoint.endpoint.ServiceName;
                    var serviceEndpoint = newPublicEndpoint.endpoint.Application;
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
                    let modalScope = $scope.$new(true);
                    modalScope.assignments = { 'ip': ip, 'value': null };
                    resourcesFactory.getPoolIPs(poolID)
                        .success(function (data) {
                            let options = [{ 'Value': 'Automatic', 'IPAddr': '' }];
                            let i, IPAddr, value;
                            //host ips
                            if (data && data.HostIPs) {
                                for (i = 0; i < data.HostIPs.length; ++i) {
                                    IPAddr = data.HostIPs[i].IPAddress;
                                    value = 'Host: ' + IPAddr + ' - ' + data.HostIPs[i].InterfaceName;
                                    options.push({ 'Value': value, 'IPAddr': IPAddr });
                                    modalScope.assignments.ip.Application = ip.Application;

                                    // set the default value to the currently assigned value
                                    if (modalScope.assignments.ip.IPAddress === IPAddr) {
                                        modalScope.assignments.value = options[options.length - 1];
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
                                    if (modalScope.assignments.ip.IPAddr === IPAddr) {
                                        modalScope.assignments.value = options[options.length - 1];
                                    }
                                }
                            }

                            //default to automatic if necessary
                            if (!modalScope.assignments.value) {
                                modalScope.assignments.value = options[0];

                            }
                            modalScope.assignments.options = options;

                            $modalService.create({
                                templateUrl: "assign-ip.html",
                                model: modalScope,
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

                                                var serviceID = modalScope.assignments.ip.ServiceID;
                                                var IP = modalScope.assignments.value.IPAddr;
                                                resourcesFactory.assignIP(serviceID, IP)
                                                    .success(function (data, status) {
                                                        $notification.create("Added IP", data.Detail).success();
                                                        $scope.currentService.fetchAddresses(true);
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
                    if($scope.currentService && $scope.currentService.exportedServiceEndpoints){
                        return $scope.currentService.exportedServiceEndpoints.length > 0;
                    } else {
                        return false;
                    }
                };

                $scope.publicEndpointProtocol = function (publicEndpoint) {
                    if ($scope.getEndpointType(publicEndpoint) === "vhost") {
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
                    return { 'padding-left': (24 * depth) + "px" };
                };

                // sets a service to start, stop or restart state
                $scope.setServiceState = function (service, state, skipChildren) {
                    // service[state]() ends up translating to something like
                    // service.start() or service.stop()
                    service[state](skipChildren).error(function (data, status) {
                        $notification.create("Unable to " + state + " service", data.Detail).error();
                    });
                };

                $scope.getEndpointType = function (endpoint) {
                    return endpoint.VHostName ? "vhost" : "port";
                };

                // clicks to a service's start, stop, or restart
                // button should first determine if the service has
                // children and ask the user to choose to start all
                // children or only the top service
                $scope.clickRunning = function (service, state, force) {
                    // if force, we dont need no stinkin confirmation
                    // from the user.
                    if(force){
                        $scope.setServiceState(service, state);
                        return;
                    }

                    let onStartService = function(modal){
                        // the arg here explicitly prevents child services
                        // from being started
                        $scope.setServiceState(service, state, true);
                        modal.close();
                    };
                    let onStartServiceAndChildren = function(modal){
                        $scope.setServiceState(service, state);
                        modal.close();
                    };

                    Service.countAffectedDescendants(service, state)
                        .then(count => {
                            $modalService.modals.confirmServiceStateChange(service, state,
                                count, onStartService,
                                onStartServiceAndChildren);
                        })
                        .catch(err => {
                            console.warn("couldnt get descendant count", err);
                            $modalService.modals.confirmServiceStateChange(service, state,
                                0, onStartService,
                                onStartServiceAndChildren);
                        });

                    // let the user know they gonna have to hold onto
                    // their horses for just one moment.
                    $modalService.modals.oneMoment();
                };

                $scope.clickEndpointEnable = function (publicEndpoint) {
                    if ($scope.getEndpointType(publicEndpoint) === "vhost") {
                        resourcesFactory.enableVHost(publicEndpoint.ServiceID, publicEndpoint.ServiceName, publicEndpoint.Application, publicEndpoint.VHostName)
                            .success(() => {
                                // reload the table
                                refreshEndpoints();
                            })
                            .error((data, status) => {
                                $notification.create("Enable Public Endpoint failed", data.Detail).error();
                            });
                    } else if ($scope.getEndpointType(publicEndpoint) === "port") {
                        resourcesFactory.enablePort(publicEndpoint.ServiceID, publicEndpoint.ServiceName, publicEndpoint.Application, publicEndpoint.PortAddress)
                            .success(() => {
                                // reload the table
                                refreshEndpoints();
                            })
                            .error((data, status) => {
                                $notification.create("Enable Public Endpoint failed", data.Detail).error();
                            });
                    }
                };


                $scope.clickEndpointDisable = function (publicEndpoint) {
                    if ($scope.getEndpointType(publicEndpoint) === "vhost") {
                        resourcesFactory.disableVHost(publicEndpoint.ServiceID, publicEndpoint.ServiceName, publicEndpoint.Application, publicEndpoint.VHostName)
                            .success(() => {
                                // reload the table
                                refreshEndpoints();
                            })
                            .error((data, status) => {
                                $notification.create("Disable Public Endpoint failed", data.Detail).error();
                            });
                    } else if ($scope.getEndpointType(publicEndpoint) === "port") {
                        resourcesFactory.disablePort(publicEndpoint.ServiceID, publicEndpoint.ServiceName, publicEndpoint.Application, publicEndpoint.PortAddress)
                            .success(() => {
                                // reload the table
                                refreshEndpoints();
                            })
                            .error((data, status) => {
                                $notification.create("Disable Public Endpoint failed", data.Detail).error();
                            });
                    }
                };

                $scope.clickEditContext = function (contextFileId) {
                    //edit variables (context) of current service
                    let modalScope = $scope.$new(true);

                    resourcesFactory.v2.getServiceContext(contextFileId)
                        .then(function (data) {

                            //set editor options for context editing
                            modalScope.codemirrorOpts = {
                                lineNumbers: true,
                                mode: "properties"
                            };

                            // this is the text bound to the modal texarea
                            modalScope.Context = makeEditableContext(data);

                            // now that we have the text of the file, create modal dialog
                            $modalService.create({
                                templateUrl: "edit-context.html",
                                model: modalScope,
                                title: $translate.instant("edit_context"),
                                actions: [
                                    {
                                        role: "cancel"
                                    }, {
                                        role: "ok",
                                        label: $translate.instant("btn_save"),
                                        action: function () {
                                            // disable ok button, and store the re-enable function
                                            let enableSubmit = this.disableSubmitButton();
                                            let storableContext = makeStorableContext(modalScope.Context);

                                            resourcesFactory.v2.updateServiceContext(contextFileId, storableContext)
                                                .success(function (data, status) {
                                                    $notification.create("Updated variables for", $scope.currentService.name).success();
                                                    this.close();
                                                }.bind(this))
                                                .error(function (data, status) {
                                                    this.createNotification("Update variables failed", data.Detail).error();
                                                    enableSubmit();
                                                }.bind(this));
                                        }
                                    }
                                ],
                                onShow: function () {
                                    modalScope.codemirrorRefresh = true;
                                    modalScope.$apply();
                                },
                                onHide: function () {
                                    modalScope.codemirrorRefresh = false;
                                }
                            });
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

                    return JSON.stringify(storable);
                }

                $scope.clickRemovePublicEndpoint = function (publicEndpoint) {

                    $modalService.create({
                        template: $translate.instant("remove_public_endpoint") + ": <strong>" +
                        (publicEndpoint.ServiceName ? publicEndpoint.ServiceName : "port " + publicEndpoint.PortAddress) + "</strong><br><br>",
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
                                    if ($scope.getEndpointType(publicEndpoint) === "vhost") {
                                        resourcesFactory.removeVHost(publicEndpoint.ServiceID, publicEndpoint.Application, publicEndpoint.VHostName)
                                            .success(() => {
                                                // reload the table
                                                refreshEndpoints();
                                                $notification.create("Removed Public Endpoint", publicEndpoint.Application).success();
                                            })
                                            .error((data, status) => {
                                                $notification.create("Remove Public Endpoint failed", data.Detail).error();
                                            });
                                    } else if ($scope.getEndpointType(publicEndpoint) === "port") {
                                        resourcesFactory.removePort(publicEndpoint.ServiceID, publicEndpoint.Application, publicEndpoint.PortAddress)
                                            .success(() => {
                                                // reload the table
                                                refreshEndpoints();
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

                $scope.editConfig = function (configFileId) {
                    let modalScope = $scope.$new(true);

                    // TODO - pop the modal up FIRST and show
                    // a loading animation while the request is filled
                    resourcesFactory.v2.getServiceConfig(configFileId)
                        .then(function (data) {

                            //set editor options for context editing
                            modalScope.codemirrorOpts = {
                                lineNumbers: true,
                                mode: utils.getModeFromFilename(data.Filename)
                            };

                            // this is the text bound to the modal texarea
                            angular.extend(modalScope, data);

                            // now that we have the text of the file, create modal dialog
                            $modalService.create({
                                templateUrl: "edit-config.html",
                                model: modalScope,
                                title: $translate.instant("title_edit_config") + " - " + modalScope.Filename,
                                bigModal: true,
                                actions: [
                                    {
                                        role: "cancel"
                                    }, {
                                        role: "ok",
                                        label: $translate.instant("btn_save"),
                                        action: function () {
                                            if (this.validate()) {
                                                // disable ok button, and store the re-enable function
                                                var enableSubmit = this.disableSubmitButton();

                                                resourcesFactory.v2.updateServiceConfig(configFileId, modalScope)
                                                    .success(function (data, status) {
                                                        $notification.create("Updated configuration file", data.Filename).success();
                                                        this.close();
                                                    }.bind(this))
                                                    .error(function (data, status) {
                                                        this.createNotification("Update configuration failed", data.Detail).error();
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
                                    modalScope.codemirrorRefresh = true;
                                    modalScope.$apply();
                                },
                                onHide: function () {
                                    modalScope.codemirrorRefresh = false;
                                }
                            });

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
                            query: `fields.service:${service.id} AND fields.instance:*`
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
                            query: `fields.service:${instance.model.ServiceID} AND fields.instance:${instance.model.InstanceID}`
                        }
                    };
                    appConfig.columns = ["message"];

                    return LogSearch.getURL(appConfig, globalConfig);

                };

                // updates URL and current service ID
                // which triggers UI update
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
                };

                // restart all running instances for this service
                $scope.killRunningInstances = function (app) {
                    resourcesFactory.restartService(app.ID)
                        .error((data, status) => {
                            $notification.create("Stop Service failed", data.Detail).error();
                        });
                };

                $scope.hostNames = {};
                $scope.getHostName = function (id) {
                    return $scope.hostNames[id];
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
                //      -> desendant service id
                //          -> hidden
                //          -> collapsed
                //          -> parentId
                //      -> desendant service id
                //          -> hidden
                //          -> collapsed
                //          -> parentId
                //      ...

                $scope.toggleChildren = function (service) {
                    if (!$scope.currentService) {
                        console.warn("Cannot store toggle state: no current service");
                        return;
                    }
                    if ($scope.currentTreeState[service.id].collapsed) {
                        $scope.currentTreeState[service.id].collapsed = false;

                        if (service.subservices.length) {
                            $scope.showChildren(service);
                        } else {
                            // show loading animation row
                            $scope.currentTreeState[service.id].isLoading = true;
                            $scope.flattenServicesTree();
                            service.fetchServiceChildren().then(() => {
                                // remove loading animation row
                                $scope.currentTreeState[service.id].isLoading = false;
                                $scope.flattenServicesTree();
                                $scope.currentService.updateDescendentStatuses();
                                service.subservices.forEach(sub => {
                                    $scope.currentTreeState[sub.id].parentId = service.id;
                                    $scope.currentTreeState[sub.id].hidden = false;
                                });
                            });
                        }

                    } else {
                        $scope.currentTreeState[service.id].collapsed = true;
                        $scope.flattenServicesTree();
                        $scope.currentService.updateDescendentStatuses();
                        $scope.hideChildren(service);
                    }
                };

                $scope.hideChildren = function (service) {
                    // get the state of the current service's tree
                    var treeState = $scope.currentTreeState;

                    if (service.subservices.length) {
                        service.subservices.forEach(function (child) {
                            treeState[child.id].hidden = true;
                            $scope.hideChildren(child);
                        });
                    }
                };

                $scope.showChildren = function (service) {
                    var treeState = $scope.currentTreeState;

                    if (service.subservices.length) {
                        service.subservices.forEach(function (child) {
                            treeState[child.id].hidden = false;

                            // if this child service is not marked
                            // as collapsed, show its children
                            if (!treeState[child.id].collapsed) {
                                $scope.showChildren(child);
                            }
                        });
                    }
                };

                $scope.getServiceEndpoints = function (id) {
                    let deferred = $q.defer();
                    resourcesFactory.v2.getServiceEndpoints(id)
                        .then(function (response) {
                            console.log("got service endpoints for id " + id);
                            deferred.resolve(response.data);
                        },
                        function (response) {
                            console.warn(response.status + " " + response.statusText);
                            deferred.reject(response.statusText);
                        });
                    return deferred.promise;
                };

                //we need to bring this function into scope so we can use ng-hide if an object is empty
                $scope.isEmptyObject = function (obj) {
                    return angular.equals({}, obj);
                };

                $scope.isIsvc = function (service) {
                    return service.isIsvc();
                };

                $scope.hasCurrentInstances = function () {
                    return $scope.currentService && $scope.currentService.hasInstances();
                };

                $scope.editCurrentService = function () {
                    let modalModel = $scope.$new(true);
                    // clone service for editing
                    $scope.editableService = angular.copy($scope.currentService.model);
                    modalModel.model = angular.copy($scope.currentService.model);
                    modalModel.pools = [];

                    // this modal needs to know all the pools,
                    // so axe for all the pools!
                    // TODO - cache this list?
                    (function getPools(){
                        resourcesFactory.v2.getPools()
                            .then(data => {
                                modalModel.pools = data.map(result => new Pool(result));
                            })
                            .catch(data => {
                                console.warn("Could not load pools, trying again in 1s");
                                $timeout(getPools, 1000);
                            });
                    })();

                    $modalService.create({
                        templateUrl: "edit-service.html",
                        model: modalModel,
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
                                        resourcesFactory.v2.updateService(modalModel.model.ID, modalModel.model)
                                            .success(function (data, status) {
                                                $notification.create("Updated service", modalModel.model.Name).success();
                                                update();
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
                            if (modalModel.model.InstanceLimits.Min > modalModel.model.Instances || modalModel.model.Instances === undefined) {
                                return false;
                            }
                            var err2 = utils.validateRAMThresholdLimit(modalModel.model.RAMThreshold);
                            if(err2){
                                this.createNotification("Error", err2).error();
                                return false;
                            }
                            var err = utils.validateRAMLimit(modalModel.model.RAMCommitment, modalModel.model.MemoryLimit);
                            if(err){
                                this.createNotification("Error", err).error();
                                return false;
                            }
                            if(modalModel.model.Launch !== "auto" && modalModel.model.Launch !== "manual") {
                                this.createNotification("Error", "Invalid launch mode selected").error();
                                return false;
                            }

                            return true;
                        }
                    });
                };

                // TODO - clean up magic numbers
                $scope.calculateIndent = function (depth) {
                    let offset = 1;
                    if ($scope.currentService && $scope.currentService.parent) {
                        offset = $scope.currentService.parent.depth + 2;
                    }
                    return $scope.indent(depth - offset);
                };

                $scope.restoreTreeState = function () {

                    if (!$scope.currentService) {
                        return;
                    }
                    // initialize at first call
                    if (!$scope.serviceTreeState[$scope.currentService.id]) {
                        $scope.serviceTreeState[$scope.currentService.id] = {};
                    }
                    // delete hidden services state upon re-entry
                    let treeState = $scope.serviceTreeState[$scope.currentService.id];
                    Object.keys(treeState).forEach(k => {
                        if (treeState[k].hidden) {
                            delete treeState[k];
                        }
                    });
                    $scope.serviceTreeState[$scope.currentService.id] = treeState;

                    $scope.updateTreeState();
                    $scope.currentTreeState = treeState;

                };

                $scope.updateTreeState = function () {
                    let treeState = $scope.serviceTreeState[$scope.currentService.id];
                    $scope.currentService.subservices.forEach(function recurse(service) {
                        // initialize on page load
                        if (!treeState[service.id]) {
                            treeState[service.id] = {
                                collapsed: true,
                                hidden: false
                            };
                            return;
                        }
                        // collapsed node - we're done here
                        if (treeState[service.id].collapsed === true) {
                            return;
                        }
                        // expanded node - get subservices
                        service.fetchServiceChildren()
                            .then(() => {
                                // toggle the UI row & set currentDescendents
                                $scope.flattenServicesTree();
                                // recurse for each child service
                                service.subservices.forEach(subservice => {
                                    recurse(subservice);
                                });
                            });
                    });
                };

                $scope.flattenServicesTree = function () {
                    // flatten the current service's subservices tree.
                    let rows = [];
                    let treeState = $scope.currentTreeState;
                    (function flatten(service, depth) {
                        if (!treeState[service.id]) {
                            treeState[service.id] = {
                                collapsed: true,
                                hidden: false
                            };
                        }
                        let rowState = treeState[service.id];
                        let rowItem = {
                            service: service,
                            depth: depth,
                            collapsed: rowState.collapsed,
                            hidden: rowState.hidden
                        };
                        rows.push(rowItem);

                        // if expanding node, add dummy row for loading animation
                        if (rowState.isLoading) {
                            let loaderRow = {
                                isDummy : true
                            };
                            rows.push(loaderRow);
                        }

                        if (service.subservices.length) {
                            $filter('orderBy')(service.subservices, 'name')
                                .forEach(svc => flatten(svc, depth + 1));
                        }
                    })($scope.currentService, 0);

                    // rows[0] is always the top level service, so slice that off
                    $scope.currentDescendents = rows.slice(1);
                };

                $scope.fetchBreadcrumbs = function () {
                    resourcesFactory.v2.getServiceAncestors($scope.currentService.id)
                        .then(current => {
                            $scope.breadcrumbs = makeCrumbs(current);
                        },
                        error => {
                            console.warn(error);
                        });
                };

                // constructs a new current service
                $scope.setCurrentService = function () {
                    let first = false;
                    // if this is the first time setting the service,
                    // be sure to emit "ready" after things are settled.
                    // the ready event clears the big loading jellyfish
                    if($scope.currentService === undefined){
                        first = true;
                    }

                    $scope.currentService = undefined;
                    resourcesFactory.v2.getService($scope.params.serviceId)
                        .then(function (model) {
                            $scope.currentService = new Service(model);

                            $scope.currentDescendents = [];
                            $scope.currentService.fetchServiceChildren()
                                .then(() => {
                                    $scope.restoreTreeState();
                                    $scope.flattenServicesTree();
                                    $scope.currentService.updateDescendentStatuses();
                                });

                            // sets $scope.breadcrumbs
                            $scope.fetchBreadcrumbs();

                            // fetchAll() will trigger update at completion
                            $scope.currentService.fetchAll(true);

                            // update fast-moving statuses
                            $scope.currentService.fetchAllStates();

                            if(first){
                                $scope.$root.$emit("ready");
                            }
                        });
                };

                $scope.shouldDisable = function(service, button) {
                    if (service === undefined) {
                        return false;
                    }
                    if (button === 'start') {
                        return service.desiredState === 1 ||
                            service.emergencyShutdown;
                    } else if (button === 'stop') {
                        return service.desiredState === 0;
                    } else if (button === 'restart') {
                      return service.desiredState === 0 ||
                          service.emergencyShutdown;
                    }
                    return false;
                };

                function refreshEndpoints() {
                    $scope.currentService.fetchEndpoints(true);
                }

                function update() {
                    // update service model data
                    resourcesFactory.v2.getService($scope.params.serviceId)
                        .then(function (model) {
                            $scope.currentService.update(model);
                        });

                    // update fast-moving statuses
                    $scope.currentService.fetchAllStates();
                    $scope.currentService.updateDescendentStatuses();
                }

                function init() {

                    $scope.name = "servicedetails";
                    $scope.params = $routeParams;

                    $scope.breadcrumbs = [
                        { label: 'breadcrumb_deployed', url: '/apps' }
                    ];

                    $scope.publicEndpointsTable = {
                        sorting: {
                            ServiceName: "asc"
                        },
                        searchColumns: ['ServiceName', 'Application', 'Protocol' ]
                    };
                    $scope.ipsTable = {
                        sorting: {
                            ServiceName: "asc"
                        },
                        searchColumns: ['ServiceName', 'PoolID', 'IPAddress', 'Port']
                    };
                    $scope.configTable = {
                        sorting: {
                            Filename: "asc"
                        },
                        searchColumns: ['Filename']
                    };
                    $scope.instancesTable = {
                        sorting: {
                            "model.InstanceID": "asc"
                        },
                        searchColumns: ['model.InstanceID', 'model.ContainerID'],
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

                    $scope.ips = {};

                    // get host status to create a map of
                    // host id to name
                    (function getHostNames(){
                        resourcesFactory.v2.getHostStatuses()
                            .then(result => {
                                $scope.hostNames = result.reduce((acc,s) => {
                                    acc[s.HostID] = s.HostName;
                                    return acc;
                                }, {});
                            })
                            .catch(err => {
                                console.warn("Could not fetch host names, retrying in 1s", err);
                                $timeout(getHostNames, 1000);
                            });
                    })();

                    // if the current service changes, update
                    // various service controller thingies
                    $scope.$watch("params.serviceId", $scope.setCurrentService);

                    // TODO - use UI_POLL_INTERVAL
                    let intervalVal = setInterval(function () {
                        if ($scope.currentService) {
                            $scope.currentService.fetchAllStates();
                            $scope.currentService.updateDescendentStatuses();
                        }
                    }, 3000);

                    $scope.$on("$destroy", function () {
                        clearInterval(intervalVal);
                    });
                }

                // kick off controller
                init();

                function makeCrumbs(current) {

                    var crumbs = [{
                        label: current.Name,
                        itemClass: "active",
                        id: current.ID
                    }];

                    (function recurse(service) {
                        if (service) {
                            crumbs.unshift({
                                label: service.Name,
                                url: "/services/" + service.ID,
                                id: service.ID
                            });
                            recurse(service.Parent);
                        }
                    })(current.Parent);

                    crumbs.unshift({
                        label: "Applications",
                        url: "/apps"
                    });

                    return crumbs;
                }

            }]);

})();
