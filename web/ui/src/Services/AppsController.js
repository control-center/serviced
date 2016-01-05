/* globals controlplane: true */

/* AppsController
 * Displays top level apps
 */
(function() {
    'use strict';

    controlplane.controller("AppsController", [
        "$scope", "$routeParams", "$location",
        "$notification", "resourcesFactory", "authService",
        "$modalService", "$translate", "$timeout",
        "$cookies", "servicesFactory", "miscUtils",
        "ngTableParams", "$filter", "poolsFactory",
    function($scope, $routeParams, $location,
    $notification, resourcesFactory, authService,
    $modalService, $translate, $timeout,
    $cookies, servicesFactory, utils,
    NgTableParams, $filter, poolsFactory){

        // Ensure logged in
        authService.checkLogin($scope);

        // alias hostname instead of using localhost or IP
        $scope.defaultHostAlias = $location.host();

        // redirect to specific service details
        $scope.routeToService = function(id) {
            resourcesFactory.routeToService(id);
        };

        // redirect to specific pool
        $scope.routeToPool = function(id) {
            resourcesFactory.routeToPool(id);
        };

        $scope.modal_deployWizard = function() {
            // the modal occasionally won't show on page load, so we use a timeout to get around that.
            $timeout(function(){
                $('#addApp').modal('show');

                // don't auto-show this wizard again
                // NOTE: $cookies can only deal with string values
                $cookies.autoRunWizardHasRun = "true";
            });
        };

        $scope.modal_addTemplate = function() {
            $modalService.create({
                templateUrl: "add-template.html",
                model: $scope,
                title: "template_add",
                actions: [
                    {
                        role: "cancel",
                        action: function(){
                            $scope.newHost = {};
                            this.close();
                        }
                    },{
                        role: "ok",
                        label: "template_add",
                        action: function(){
                            if(this.validate()){
                                var data = new FormData();

                                $.each($("#new_template_filename")[0].files, function(key, value){
                                    data.append("tpl", value);
                                });

                                // disable ok button, and store the re-enable function
                                var enableSubmit = this.disableSubmitButton();

                                resourcesFactory.addAppTemplate(data)
                                    .success(function(data){
                                        $notification.create("Added template", data.Detail).success();
                                        refreshTemplates();
                                        this.close();
                                    }.bind(this))
                                    .error(function(data){
                                        this.createNotification("Adding template failed", data.Detail).error();
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ]
            });
        };

        // aggregate vhosts for a specified service, but
        // only if the service has changed since last request
        $scope.aggregateVHosts = utils.memoize(function(service) {
            var endPoints = [];

            service.model.Endpoints.forEach(endpoint => {
                if(endpoint.VHostList){
                    endpoint.VHostList.forEach(vHost => endPoints.push(vHost));
                }
                if(endpoint.PortList){
                    endpoint.PortList.forEach(port => endPoints.push(port));
                }
            });

            endPoints.sort();

            return endPoints;
        }, function(service){
            return service.id + service.model.DatabaseVersion;
        });

        // given an endpoint, return a url to it
        $scope.publicEndpointURL = function(publicEndpoint) {
            if ("Name" in publicEndpoint){
                var port = $location.port() === "" ? "" : ":"+$location.port();
                var host = publicEndpoint.Name.indexOf('.') === -1 ? publicEndpoint.Name + "." + $scope.defaultHostAlias : publicEndpoint.Name;
                return $location.protocol() + "://" + host + port;
            } else if ("PortNumber" in publicEndpoint){
                // Port public endpoint port listeners are always on http
                return "http://" + $scope.defaultHostAlias + ":" + publicEndpoint.PortNumber;
            }
        };

        $scope.modal_removeService = function(service) {
            $modalService.create({
                template: $translate.instant("warning_remove_service"),
                model: $scope,
                title: "remove_service",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "remove_service",
                        classes: "btn-danger submit",
                        action: function(){
                            if(this.validate()){
                                this.disableSubmitButton();

                                removeService(service)
                                    .success(() => {
                                        $notification.create("Removed App", service.name).success();
                                        this.close();
                                    })
                                    .error((data, status) => {
                                        $notification.create("Remove App failed", data.Detail).error();
                                        this.close();
                                    });
                            }
                        }
                    }
                ]
            });
        };

        $scope.startService = function(service){
            $scope.modal_startStopService(service, "start");
        };
        $scope.stopService = function(service){
            $scope.modal_startStopService(service, "stop");
        };
        $scope.modal_startStopService = function(service, status) {
            var displayStatus = utils.capitalizeFirst(status);

            $modalService.create({
                template: $translate.instant("confirm_"+ status +"_app"),
                model: $scope,
                title: displayStatus +" Services",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: displayStatus +" Services",
                        action: function(){
                            service[status]();
                            this.close();
                        }
                    }
                ]
            });
        };

        $scope.clickRunningApp = function(service, status) {
            utils.setServiceState($scope, service, status, $modalService, $translate);
        };

        $scope.modal_deleteTemplate = function(templateID){
            $modalService.create({
                template: $translate.instant("template_remove_confirm") + "<strong>"+ templateID +"</strong>",
                model: $scope,
                title: "template_remove",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "template_remove",
                        classes: "btn-danger",
                        action: function(){
                            deleteTemplate(templateID)
                                .success(() => {
                                    $notification.create("Removed Template", templateID).success();
                                    this.close();
                                })
                                .error((data, status) => {
                                    $notification.create("Remove Template failed", data.Detail).error();
                                });
                        }
                    }
                ]
            });
        };


        /*
         * PRIVATE FUNCTIONS
         */
        function refreshTemplates(){
            resourcesFactory.getAppTemplates().success(function(templates) {
                $scope.templates.data = utils.mapToArr(templates);
            });
        }

        // poll for apps that are being deployed
        $scope.deployingServices = [];
        var lastPollResults = 0;
        function getDeploying(){
            resourcesFactory.getDeployingTemplates().success(function(data) {
                if(data){
                    $scope.deployingServices = data;
                }

                //if we have fewer results than last poll, we need to refresh our table
                //TODO - better checking for deploying apps
                if(lastPollResults > $scope.deployingServices.length){
                    servicesFactory.update();
                }
                lastPollResults = $scope.deployingServices.length;
            });
        }

        function removeService(service) {
            return resourcesFactory.removeService(service.id)
                .success(function(){
                    // NOTE: this is here because services are
                    // incrementally updated, which makes it impossible
                    // to determine if a service has been removed
                    // TODO - once the backend notifies on deleted
                    // services, this should be removed
                    // FIXME - should not modify servicesFactory's
                    // objects!
                    for(var i = 0; i < $scope.apps.length; i++){
                        // find the removed service and remove it
                        if($scope.apps[i].id === service.id){
                            $scope.apps.splice(i, 1);
                            return;
                        }
                    }
                });
        }

        function deleteTemplate(templateID){
            return resourcesFactory.removeAppTemplate(templateID)
                .success(refreshTemplates);
        }

        // init stuff for this controller
        function init(){
            if(utils.needsHostAlias($location.host())){
                resourcesFactory.getHostAlias().success(function(data) {
                    $scope.defaultHostAlias = data.hostalias;
                });
            }

            // configure tables
            // TODO - move table config to view/directive
            $scope.breadcrumbs = [
                { label: 'breadcrumb_deployed', itemClass: 'active' }
            ];

            $scope.servicesTable = {
                sorting: {
                    name: "asc"
                },
                getData: function(data, params) {
                    // use built-in angular filter
                    var orderedData = params.sorting() ?
                        $filter('orderBy')(data, params.orderBy()) :
                        data;

                    if(!orderedData){
                        return;
                    }

                    // mark any deploying services so they can be treated differently
                    orderedData.forEach(function(service){
                        service.deploying = false;
                        $scope.deployingServices.forEach(function(deploying){
                            if(service.model.DeploymentID === deploying.DeploymentID){
                                service.deploying = true;
                            }
                        });
                    });

                    return orderedData;
                },
                watchExpression: function(){
                    // TODO - check $scope.deployingServices as well
                    return servicesFactory.lastUpdate;
                }
            };

            $scope.templates = { data: [] };
            // table config
            $scope.templatesTable = {
                sorting: {
                    Name: "asc"
                }
            };

            // Get a list of templates
            refreshTemplates();

            // check for deploying apps
            getDeploying();

            // start polling for apps
            servicesFactory.activate();
            servicesFactory.update().then(function(){
                // locally bind top level service list
                $scope.apps = servicesFactory.serviceTree;

                // if only isvcs are deployed, and this is the first time
                // running deploy wizard, show the deploy apps modal
                if(!$cookies.autoRunWizardHasRun && $scope.apps.length === 1){
                    $scope.modal_deployWizard();
                }
            });

            // deploy wizard needs updated pools
            poolsFactory.activate();

            //register polls
            resourcesFactory.registerPoll("deployingApps", getDeploying, 3000);
            resourcesFactory.registerPoll("templates", refreshTemplates, 5000);

            //unregister polls on destroy
            $scope.$on("$destroy", function(){
                resourcesFactory.unregisterAllPolls();
            });
        }

        // kick this controller off
        init();

        $scope.$on("$destroy", function(){
            resourcesFactory.unregisterAllPolls();
            servicesFactory.deactivate();
            poolsFactory.deactivate();
        });
    }]);
})();
