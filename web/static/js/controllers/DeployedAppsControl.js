function DeployedAppsControl($scope, $routeParams, $location, $notification, resourcesService, $serviceHealth, authService, $modalService, $translate, $timeout, $cookies, $servicesService){
    // Ensure logged in
    authService.checkLogin($scope);

    //constantly poll for apps that are in the process of being deployed so we can alert the user
    $scope.deployingServices = [];
    var lastPollResults = 0;
    var pollDeploying = function(){
        resourcesService.get_active_templates(function(data) {
            if(data === "null"){
                $scope.services.deploying = [];
            }else{
                $scope.services.deploying = data;
            }

            //if we have fewer results than last poll, we need to refresh our table
            if(lastPollResults > $scope.services.deploying.length){
                $servicesService.update();
            }
            lastPollResults = $scope.services.deploying.length;
        });
    };
    $scope.$on("$destroy", function(){
        resourcesService.unregisterAllPolls();
    });
    $scope.name = "apps";
    $scope.params = $routeParams;
    $scope.resourcesService = resourcesService;

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

                            resourcesService.add_app_template(data)
                                .success(function(data, status){
                                    $notification.create("Added template", data.Detail).success();
                                    resourcesService.get_app_templates(false, refreshTemplates);
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

        if (app.Endpoints) {
            for (var i in app.Endpoints) {
                var endpoint = app.Endpoints[i];
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
        return location.protocol + "//" + host + port
    };

    $scope.clickRemoveService = function(app) {
        $scope.appToRemove = app;
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
                            $scope.remove_service();
                            this.close();
                        }
                    }
                }
            ]
        });
    };

    $scope.remove_service = function() {
        if (!$scope.appToRemove) {
            console.log('No selected service to remove');
            return;
        }
        var id = $scope.appToRemove.ID;
        resourcesService.remove_service(id, function() {
            delete $scope.appToRemove;
            var i = 0, newServices = [];

            // build up a new services array containing all the services
            // except the one we just deleted
            for (i=0;i<$scope.services.data.length;i++) {
                if ($scope.services.data[i].ID != id) {
                    newServices.push($scope.services.data[i]);
                }
            }
            $scope.services.data = newServices;
        });
    };

    $scope.clickRunning = function(app, status, resourcesService) {
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

    var setupNewService = function() {
        $scope.newService = {
            poolID: 'default',
            ParentServiceID: '',
            DesiredState: 1,
            Launch: 'auto',
            Instances: 1,
            Description: '',
            ImageID: ''
        };
    };
    $scope.click_secondary = function(navlink) {
        if (navlink.path) {
            $location.path(navlink.path);
        }
        else if (navlink.modal) {
            $(navlink.modal).modal('show');
        }
        else {
            console.log('Unexpected navlink: %s', JSON.stringify(navlink));
        }
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
                        resourcesService.delete_app_template(templateID, refreshTemplates);
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
        resourcesService.get_app_templates(false, function(templatesMap) {
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
    $servicesService.update().then(function update(){
        $scope.services.data = $servicesService.serviceTree;

        // if only isvcs are deployed, and this is the first time
        // running deploy wizard, show the deploy apps modal
        if(!$cookies.autoRunWizardHasRun && $scope.services.data.length === 1){
            $scope.modalAddApp();
        }
    });

    //register polls
    resourcesService.registerPoll("deployingApps", pollDeploying, 3000);
}
