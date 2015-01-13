/* AppsController
 * Displays top level apps
 */
(function() {
    'use strict';

    controlplane.controller("AppsController", ["$scope", "$routeParams", "$location", "$notification", "resourcesFactory", "authService", "$modalService", "$translate", "$timeout", "$cookies", "servicesFactory",
    function($scope, $routeParams, $location, $notification, resourcesFactory, authService, $modalService, $translate, $timeout, $cookies, servicesFactory){
        // Ensure logged in
        authService.checkLogin($scope);

        //constantly poll for apps that are in the process of being deployed so we can alert the user
        $scope.deployingServices = [];
        var lastPollResults = 0;
        var pollDeploying = function(){
            resourcesFactory.get_active_templates(function(data) {
                if(data === "null"){
                    $scope.services.deploying = [];
                }else{
                    $scope.services.deploying = data;
                }

                //if we have fewer results than last poll, we need to refresh our table
                if(lastPollResults > $scope.services.deploying.length){
                    servicesFactory.update();
                }
                lastPollResults = $scope.services.deploying.length;
            });
        };
        $scope.$on("$destroy", function(){
            resourcesFactory.unregisterAllPolls();
        });
        $scope.name = "apps";
        $scope.params = $routeParams;
        $scope.resourcesFactory = resourcesFactory;

        $scope.defaultHostAlias = location.hostname;
        var re = /\b(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b/;
        if (re.test(location.hostname) || location.hostname == "localhost") {
            $.getJSON("/hosts/defaultHostAlias", "", function(data) {
                $scope.defaultHostAlias = data.hostalias;
            });
        }

        $scope.breadcrumbs = [
            { label: 'breadcrumb_deployed', itemClass: 'active' }
        ];

        $scope.services = buildTable('PoolID', [
            { id: 'Name', name: 'deployed_tbl_name'},
            { id: 'Description', name: 'deployed_tbl_description'},
            { id: 'Health', name: 'health_check', hideSort: true},
            { id: 'DeploymentID', name: 'deployed_tbl_deployment_id'},
            { id: 'PoolID', name: 'deployed_tbl_pool'},
            { id: 'VirtualHost', name: 'vhost_names', hideSort: true}
        ]);

        $scope.templates = buildTable('Name', [
            { id: 'Name', name: 'template_name'},
            { id: 'ID', name: 'template_id'},
            { id: 'Description', name: 'template_description'}
        ]);

        $scope.click_app = function(id) {
            $location.path('/services/' + id);
        };

        $scope.click_pool = function(id) {
            $location.path('/pools/' + id);
        };

        $scope.modalAddApp = function() {
            // the modal occasionally won't show on page load, so we use a timeout to get around that.
            $timeout(function(){$('#addApp').modal('show');});

            // don't auto-show this wizard again
            // NOTE: $cookies can only deal with string values
            $cookies.autoRunWizardHasRun = "true";
        };

        $scope.modalAddTemplate = function() {
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
                                    .success(function(data, status){
                                        $notification.create("Added template", data.Detail).success();
                                        resourcesFactory.get_app_templates(false, refreshTemplates);
                                        this.close();
                                    }.bind(this))
                                    .error(function(data, status){
                                        this.createNotification("Adding template failed", data.Detail).error();
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ]
            });
        };

        // given a service application find all of it's virtual host names
        $scope.collect_vhosts = function(app) {
            var vhosts = [];

            if (app.service.Endpoints) {
                for (var i in app.service.Endpoints) {
                    var endpoint = app.service.Endpoints[i];
                    if (endpoint.VHosts) {
                        for ( var j in endpoint.VHosts) {
                            vhosts.push( endpoint.VHosts[j] );
                        }
                    }
                }
            }

            vhosts.sort();
            return vhosts;
        };

        // given a vhost, return a url to it
        $scope.vhost_url = function(vhost) {
            var port = location.port === "" ? "" : ":"+location.port;
            var host = vhost.indexOf('.') === -1 ? vhost + "." + $scope.defaultHostAlias : vhost;
            return location.protocol + "//" + host + port;
        };

        $scope.clickRemoveService = function(app) {
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
                                $scope.remove_service(app);
                                this.close();
                            }
                        }
                    }
                ]
            });
        };

        $scope.remove_service = function(service) {
            resourcesFactory.remove_service(service.id, function(){
                // TODO - once the backend updates deleted
                // services, this should be removed
                // HACK - should not modify servicesFactory's
                // objects!
                for(var i = 0; i < $scope.services.data.length; i++){
                    // find the removed service and remove it
                    if($scope.services.data[i].id === service.id){
                        $scope.services.data.splice(i, 1);
                        return;
                    }
                }
            });
        };

        $scope.clickRunning = function(app, status, resourcesFactory) {
            var displayStatus = capitalizeFirst(status);

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
                            // TODO - verify status is valid
                            app[status]();
                            this.close();
                        }
                    }
                ]
            });
        };

        $scope.deleteTemplate = function(templateID){
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
                            resourcesFactory.delete_app_template(templateID, refreshTemplates);
                            this.close();
                        }
                    }
                ]
            });
        };

        function capitalizeFirst(str){
            return str.slice(0,1).toUpperCase() + str.slice(1);
        }

        function refreshTemplates(){
            resourcesFactory.get_app_templates(false, function(templatesMap) {
                var templates = [];
                for (var key in templatesMap) {
                    var template = templatesMap[key];
                    template.Id = key;
                    templates[templates.length] = template;
                }
                $scope.templates.data = templates;
            });
        }

        // Get a list of templates
        refreshTemplates();

        // Get a list of deployed apps
        servicesFactory.update().then(function update(){
            $scope.services.data = servicesFactory.serviceTree;

            // if only isvcs are deployed, and this is the first time
            // running deploy wizard, show the deploy apps modal
            if(!$cookies.autoRunWizardHasRun && $scope.services.data.length === 1){
                $scope.modalAddApp();
            }
        });

        //register polls
        resourcesFactory.registerPoll("deployingApps", pollDeploying, 3000);
    }]);
})();
