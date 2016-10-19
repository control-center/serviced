/* globals controlplane: true */

/* ServiceDetailsController
 * Displays details of a specific service
 */
(function () {
    'use strict';

    // share angular services outside of angular context
    let $notification, serviceHealth, $q, resourcesFactory, utils;

    controlplane.controller("ServiceDetailsController", [
            "$scope", "$q", "$routeParams", "$location", "resourcesFactory",
            "authService", "$modalService", "$translate", "$notification",
            "$timeout", "miscUtils", "hostsFactory", "$serviceHealth", "Service",
            "poolsFactory", "CCUIState", "$cookies", "areUIReady", "LogSearch",
            "$filter",
            function ($scope, _$q, $routeParams, $location, _resourcesFactory,
                authService, $modalService, $translate, _$notification,
                $timeout, _utils, hostsFactory, _serviceHealth, Service,
                poolsFactory, CCUIState, $cookies, areUIReady, LogSearch,
                $filter) {

                // api access via angular context
                $notification = _$notification;
                serviceHealth = _serviceHealth;
                $q = _$q;
                resourcesFactory = _resourcesFactory;
                utils = _utils;

                // Ensure logged in
                authService.checkLogin($scope);
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
                                                        update();
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
                $scope.clickRunning = function (service, state) {
                    resourcesFactory.v2.getDescendantCounts(service.id)
                      .success(function (data, status) {
                        var childCount = 0;
                        switch (state) {
                          case "start":
                            // When starting, we only care about autostart
                            // services that are currently stopped
                            if (data.auto) {
                              childCount += data.auto["0"] || 0;
                            }
                            break;
                          case "restart":
                          case "stop":
                            // When stopping or restarting, we care about
                            // running services that are either manual or
                            // autostart
                            if (data.auto) {
                              childCount += data.auto["1"] || 0;
                            }
                            if (data.manual) {
                              childCount += data.manual["1"] || 0;
                            }
                            break;
                        }
                        if (childCount > 0) {
                            // if the service has affected children, check if the user
                            // wants to start just the service, or the service and children
                            $scope.modal_confirmSetServiceState(service, state, childCount);
                        } else {
                            // if no children, just start the service
                            $scope.setServiceState(service, state);
                        }
                      }.bind(this))
                      .error(function (data, status) {
                          console.log("unable to obtain descendant counts");
                          $scope.modal_confirmSetServiceState(service, state, "unknown");
                      }.bind(this));
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
                                                        $notification.create("Updated configuation file", data.Filename).success();
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
                    if (!$scope.currentService) {
                        console.warn("Cannot store toggle state: no current service");
                        return;
                    }
                    if ($scope.currentTreeState[service.id].collapsed) {
                        $scope.currentTreeState[service.id].collapsed = false;

                        if (service.subservices.length) {
                            $scope.showChildren(service);
                        } else {
                            service.fetchServiceChildren().then(() => {
                                $scope.flattenServicesTree();
                                $scope.currentService.updateDescendentStatuses();
                            });
                        }
                    } else {
                        $scope.currentTreeState[service.id].collapsed = true;
                        $scope.flattenServicesTree();
                        $scope.currentService.updateDescendentStatuses();
                        $scope.hideChildren(service);
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

                    // clone service for editing
                    $scope.editableService = angular.copy($scope.currentService.model);

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
                                        resourcesFactory.v2.updateService($scope.editableService.ID, $scope.editableService)
                                            .success(function (data, status) {
                                                $notification.create("Updated service", $scope.editableService.Name).success();
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
                            if ($scope.editableService.InstanceLimits.Min > $scope.editableService.Instances || $scope.editableService.Instances === undefined) {
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

                $scope.setCurrentTreeState = function () {
                    $scope.serviceTreeState[$scope.currentService.id] = {};
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
                        // TODO - merge rather than overwrite to avoid
                        // causing the entire tree to bounce
                        let rowItem = {
                            service: service,
                            depth: depth,
                            collapsed: rowState.collapsed,
                            hidden: rowState.hidden
                        };
                        rows.push(rowItem);
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
                    $scope.currentService = undefined;
                    resourcesFactory.v2.getService($scope.params.serviceId)
                        .then(function (model) {
                            $scope.currentService = new Service(model);

                            $scope.currentDescendents = [];
                            $scope.currentService.fetchServiceChildren()
                                .then(() => {
                                    $scope.flattenServicesTree();
                                    $scope.currentService.updateDescendentStatuses();
                                });

                            // sets $scope.breadcrumbs
                            $scope.fetchBreadcrumbs();

                            // update serviceTreeState
                            $scope.setCurrentTreeState();

                            // property for view to bind for tree state NOTE: WHA????
                            $scope.currentTreeState = $scope.serviceTreeState[$scope.currentService.id];

                            // fetchAll() will trigger update at completion
                            $scope.currentService.fetchAll(true);

                            // update fast-moving statuses
                            $scope.currentService.fetchAllStates();
                        });
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

                    $scope.ips = {};

                    // pools are needed for edit service dialog
                    $scope.pools = poolsFactory.poolList;

                    // if the current service changes, update
                    // various service controller thingies
                    $scope.$watch("params.serviceId", $scope.setCurrentService);

                    hostsFactory.activate();
                    hostsFactory.update();

                    // TODO - use UI_POLL_INTERVAL
                    let intervalVal = setInterval(function () {
                        if ($scope.currentService) {
                            $scope.currentService.fetchAllStates();
                            $scope.currentService.updateDescendentStatuses();
                        }
                    }, 3000);

                    poolsFactory.activate();
                    poolsFactory.update();

                    $scope.$on("$destroy", function () {
                        clearInterval(intervalVal);
                        // servicesFactory.deactivate();
                        hostsFactory.deactivate();
                        poolsFactory.deactivate();
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
