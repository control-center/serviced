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
    function($scope, $routeParams, $location, $notification, resourcesFactory, authService, $modalService, $translate, $timeout, $cookies, servicesFactory, utils){
        // Ensure logged in
        authService.checkLogin($scope);

        // alias hostname instead of using localhost or IP
        $scope.defaultHostAlias = $location.host();

        // redirect to specific service details
        $scope.routeToService = function(id) {
            $location.path('/services/' + id);
        };

        // redirect to specific pool
        $scope.routeToPool = function(id) {
            $location.path('/pools/' + id);
        };

        $scope.modal_deployWizard = function() {
            // the modal occasionally won't show on page load, so we use a timeout to get around that.
            // TODO - use a separate controller for deploy wizard
            $timeout(function(){
                $('#addApp').modal('show');
            });

            // don't auto-show this wizard again
            // NOTE: $cookies can only deal with string values
            $cookies.autoRunWizardHasRun = "true";
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

                                resourcesFactory.add_app_template(data)
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
            var vHosts = [];

            service.model.Endpoints.forEach(endpoint => {
                if(endpoint.VHosts){
                    endpoint.VHosts.forEach(vHost => vHosts.push(vHost));
                }
            });

            vHosts.sort();

            return vHosts;
        }, function(service){
            return service.id + service.model.DatabaseVersion;
        });

        // given a vhost, return a url to it
        $scope.createVHostURL = function(vhost) {
            var port = $location.port() === "" ? "" : ":"+$location.port();
            var host = vhost.indexOf('.') === -1 ? vhost + "." + $scope.defaultHostAlias : vhost;
            return $location.protocol() + "//" + host + port;
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
                        classes: "btn-danger",
                        action: function(){
                            if(this.validate()){
                                removeService(service);
                                this.close();
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
                            deleteTemplate(templateID);
                            this.close();
                        }
                    }
                ]
            });
        };

        $scope.tableSort = function(service){
            var sort = $scope.services.sort;
            if(sort[0] === "-"){
                sort = sort.substr(1);
            }
            return service.model[sort];
        };



        /*
         * PRIVATE FUNCTIONS
         */
        function refreshTemplates(){
            resourcesFactory.get_app_templates(false, function(templates) {
                $scope.templates.data = utils.mapToArr(templates);
            });
        }

        // poll for apps that are being deployed
        $scope.deployingServices = [];
        var lastPollResults = 0;
        function getDeploying(){
            resourcesFactory.get_active_templates(function(data) {
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
            resourcesFactory.remove_service(service.id, function(){
                // TODO - once the backend updates deleted
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
            resourcesFactory.delete_app_template(templateID, refreshTemplates);
        }

        // init stuff for this controller
        function init(){
            // check is location.hostname is an IP
            var re = /\b(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b/;
            if (re.test($location.host()) || $location.host() === "localhost") {
                // request host alias from API
                $.getJSON("/hosts/defaultHostAlias", "", function(data) {
                    $scope.defaultHostAlias = data.hostalias;
                });
            }

            // configure tables
            // TODO - move table config to view/directive
            $scope.breadcrumbs = [
                { label: 'breadcrumb_deployed', itemClass: 'active' }
            ];

            $scope.services = utils.buildTable('PoolID', [
                { id: 'Name', name: 'deployed_tbl_name'},
                { id: 'Description', name: 'deployed_tbl_description'},
                { id: 'Health', name: 'health_check', hideSort: true},
                { id: 'DeploymentID', name: 'deployed_tbl_deployment_id'},
                { id: 'PoolID', name: 'deployed_tbl_pool'},
                { id: 'VirtualHost', name: 'vhost_names', hideSort: true}
            ]);

            $scope.templates = utils.buildTable('Name', [
                { id: 'Name', name: 'template_name'},
                { id: 'ID', name: 'template_id'},
                { id: 'Description', name: 'template_description'}
            ]);

            // Get a list of templates
            refreshTemplates();

            // check for deploying apps
            getDeploying();

            // start polling for apps
            servicesFactory.activate();
            servicesFactory.update().then(function(){
                // if only isvcs are deployed, and this is the first time
                // running deploy wizard, show the deploy apps modal
                if(!$cookies.autoRunWizardHasRun && $scope.services.data.length === 1){
                    $scope.modalAddApp();
                }

                // locally bind top level service list
                $scope.apps = servicesFactory.serviceTree;
            });

            //register polls
            resourcesFactory.registerPoll("deployingApps", getDeploying, 3000);
            resourcesFactory.registerPoll("templates", refreshTemplates, 3000);

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
        });
    }]);
})();
