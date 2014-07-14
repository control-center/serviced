function DeployedAppsControl($scope, $routeParams, $location, $notification, resourcesService, $serviceHealth, authService, $modalService, $translate) {
    // Ensure logged in
    authService.checkLogin($scope);
    $scope.name = "apps";
    $scope.params = $routeParams;
    $scope.servicesService = resourcesService;

    $scope.defaultHostAlias = location.hostname;
    var re = /\b(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b/
    if (re.test(location.hostname) || location.hostname == "localhost") {
        $.getJSON("/hosts/defaultHostAlias", "", function(data) {
            $scope.defaultHostAlias = data.hostalias;
        });
    }

    $scope.breadcrumbs = [
        { label: 'breadcrumb_deployed', itemClass: 'active' }
    ];

    $scope.services = buildTable('poolID', [
        { id: 'Name', name: 'deployed_tbl_name'},
        { id: 'Health', name: 'health_check'},
        { id: 'Deployment', name: 'deployed_tbl_deployment'},
        { id: 'Id', name: 'deployed_tbl_deployment_id'},
        { id: 'poolID', name: 'deployed_tbl_pool'},
        { id: 'VirtualHost', name: 'vhost_names'}
    ]);

    $scope.click_app = function(id) {
        $location.path('/services/' + id);
    };

    $scope.click_pool = function(id) {
        $location.path('/pools/' + id);
    };

    $scope.modalAddApp = function() {
        $('#addApp').modal('show');
    };

    // given a service application find all of it's virtual host names
    $scope.collect_vhosts = function( app) {
        var vhosts = [];
        var vhosts_definitions = aggregateVhosts( app);
        for ( var i in vhosts_definitions) {
            vhosts.push( vhosts_definitions[i].Name);
        }
        return vhosts;
    };

    // given a vhost, return a url to it
    $scope.vhost_url = function( vhost) {
        var port = location.port === "" ? "" : ":"+location.port;
        return location.protocol + "//" + vhost + "." + $scope.defaultHostAlias + port;
    };

    $scope.clickRemoveService = function(app) {
        $scope.appToRemove = app;
        $modalService.create({
            template: $translate("warning_remove_service"),
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
                            // NOTE: should wait for success before closing
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

    $scope.clickRunning = function(app, status, servicesService){
        var displayStatus = capitalizeFirst(status);

        $modalService.create({
            template: $translate("confirm_"+ status +"_app"),
            model: $scope,
            title: displayStatus +" Services",
            actions: [
                {
                    role: "cancel"
                },{
                    role: "ok",
                    label: displayStatus +" Services",
                    action: function(){
                        toggleRunning(app, status, servicesService);
                        this.close();
                    }
                }
            ]
        });
    };

    // Get a list of deployed apps
    refreshServices($scope, resourcesService, false, function(){
        $serviceHealth.update();
    });

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

    if ($scope.dev) {
        setupNewService();
        $scope.add_service = function() {
            resourcesService.add_service($scope.newService, function() {
                refreshServices($scope, resourcesService, false);
                setupNewService();
            });
        };
        $scope.secondarynav.push({ label: 'btn_add_service', modal: '#addService' });
    }

    function capitalizeFirst(str){
        return str.slice(0,1).toUpperCase() + str.slice(1);
    }
}
