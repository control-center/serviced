/* globals controlplane: true */

/* AppsController
 * Displays top level apps
 */
(function() {
    'use strict';

    controlplane.controller("AppsController", [
        "$rootScope", "$scope", "$routeParams", "$location",
        "$notification", "resourcesFactory", "authService",
        "$modalService", "$translate", "$timeout",
        "$cookies", "miscUtils",
        "ngTableParams", "$filter",
        "Service","InternalService", "$q",
    function($rootScope, $scope, $routeParams, $location,
    $notification, resourcesFactory, authService,
    $modalService, $translate, $timeout,
    $cookies, utils,
    NgTableParams, $filter,
    Service, InternalService, $q){

        // Ensure logged in
        authService.checkLogin($scope);

        // alias hostname instead of using localhost or IP
        $scope.defaultHostAlias = $location.host();

        // redirect to specific service details
        $scope.routeToService = function(service) {
            if (service.isIsvc()) {
                resourcesFactory.routeToInternalServices();
            } else {
                resourcesFactory.routeToService(service.id);
            }
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
                $cookies.put("autoRunWizardHasRun","true");
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
                                        refreshApps().then(() => {
                                            $notification.create("Removed App", service.name).success();
                                        });
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

        // sets a service to start, stop or restart state
        $scope.setServiceState = function(service, state, skipChildren){
            service[state](skipChildren).error(function(data, status){
                $notification.create("Unable to " + state + " service", data.Detail).error();
            });
        };

        // clicks to a service's start, stop, or restart
        // button should first determine if the service has
        // children and ask the user to choose to start all
        // children or only the top service
        $scope.clickRunning = function(service, state){
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

        function refreshApps() {
            return fetchApps()
                .then(fillOutApps)
                .then(fetchServicesHealth)
                .then(fetchInternalServicesHealth)
                .then(() => touch())
                .catch((error) => console.warn(error));
        }

        function fetchApps() {
            return $q.all([fetchInternalService(), fetchServices()])
                .then(results => {
                    $scope.apps = [];

                    var isvc = results[0];
                    $scope.apps.push(isvc);

                    var services = results[1];
                    services.forEach(s => {
                        $scope.apps.push(s);
                    });
                });
        }

        function fetchChildEndpoints(app) {
            return resourcesFactory.v2.getServiceChildPublicEndpoints(app.id).then(data => {
                app.publicEndpoints = data;
            });
        }

        function fillOutApps() {
            let promises = [];
            $scope.apps.forEach(a => {
                if (a.isIsvc()) {
                    promises.push(() => a.fetchInstances());
                } else {
                    promises.push(fetchChildEndpoints(a));
                }
            });

            return $q.all(promises);
        }

        function fetchServices() {
            let deferred = $q.defer();
            resourcesFactory.v2.getTenants().then( data => {
                    var services = data.map(s => new Service(s));
                    deferred.resolve(services);
                },
                error => {
                    console.warn(error);
                    deferred.reject();
                });
            return deferred.promise;
        }

        function fetchServicesHealth() {
            if (!$scope.apps) { return; }

            $scope.apps.forEach(app => {
                if (!app.isIsvc()) {
                    app.fetchAllStates();
                }
            });
        }

        function fetchInternalService() {
            let deferred = $q.defer();
            resourcesFactory.v2.getInternalServices()
                .then(data => {
                    let parent = data.find(i => !i.Parent);
                    if (parent) {
                        var internalServices = new InternalService(parent);
                        deferred.resolve(internalServices);
                    } else {
                        deferred.reject("Parent not found");
                    }
                },
                error => {
                    console.warn(error);
                    deferred.reject();
                });
            return deferred.promise;
        }


        function fetchInternalServicesHealth() {
            $scope.apps.forEach(app => {
                if (app.isIsvc()) {
                    app.fetchInstances().then(() => {
                        resourcesFactory.v2.getInternalServiceStatuses([app.id]).then(data => {
                            let statusMap = data.reduce((map, s) => {
                                map[s.ServiceID] = s;
                                return map;
                            }, {});

                            app.updateStatus(statusMap[app.id]);
                        });
                    });
                }
            });
        }


        function fetchGraphConfigs() {
          resourcesFactory.getStorage().then(data => {
            for (let [n, s] of data.entries()) {
              s.MonitoringProfile.GraphConfigs.forEach(g => {
                if (g.id === "storage.pool.data.usage") {
                  // in the case where we have more than one entry in storage
                  // we need our graph ids to be unique, so concat their index
                  g.id += "." + String(n);
                  $scope.graphConfigs.push(g);
                }
              });
            }
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
                    refreshApps();
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
                            touch();
                            return;
                        }
                    }
                });
        }

        function deleteTemplate(templateID){
            return resourcesFactory.removeAppTemplate(templateID)
                .success(refreshTemplates);
        }

        var lastUpdate;
        function touch() {
            lastUpdate = new Date().getTime();
        }

        // init stuff for this controller
        function init(){
            touch();

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
                searchColumns: ['name','model.Description', 'model.DeploymentID', 'model.PoolID'],
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
                    return lastUpdate;
                }
            };

            $scope.templates = { data: [] };
            // table config
            $scope.templatesTable = {
                sorting: {
                    Name: "asc"
                },
                searchColumns: ['Name','ID', 'Description']
            };

            // get configurations for graphs
            $scope.graphConfigs = [];
            fetchGraphConfigs();

            // Get a list of templates
            refreshTemplates();

            // check for deploying apps
            getDeploying();

            refreshApps().then(() => {
                $scope.$emit("ready");

                // if only isvcs are deployed, and this is the first time
                // running deploy wizard, show the deploy apps modal
                if(!$cookies.get("autoRunWizardHasRun") && $scope.apps.length === 1){
                    $scope.modal_deployWizard();
                }
            });

            //register polls
            resourcesFactory.registerPoll("deployingApps", getDeploying, 3000);
            resourcesFactory.registerPoll("templates", refreshTemplates, 5000);
            resourcesFactory.registerPoll("internalServiceHealth", fetchInternalServicesHealth, 3000);
            resourcesFactory.registerPoll("appHealth", fetchServicesHealth, 3000);

            //unregister polls on destroy
            $scope.$on("$destroy", function(){
                resourcesFactory.unregisterAllPolls();
            });

            // be sure to update apps table after deploywiz
            // finishes its dark magicks
            $rootScope.$on("wizard.deployed", () => {
                refreshApps();
            });
        }

        // kick this controller off
        init();

        $scope.$on("$destroy", function(){
            resourcesFactory.unregisterAllPolls();
        });
    }]);
})();
