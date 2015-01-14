/* PoolsControl
 * Displays list of pools
 */
(function() {
    'use strict';

    controlplane.controller("PoolsController", ["$scope", "$routeParams", "$location", "$filter", "$timeout", "resourcesFactory", "authService", "$modalService", "$translate", "$notification",
    function($scope, $routeParams, $location, $filter, $timeout, resourcesFactory, authService, $modalService, $translate, $notification){
        // Ensure logged in
        authService.checkLogin($scope);

        $scope.name = "pools";
        $scope.params = $routeParams;
        $scope.newPool = {};

        $scope.breadcrumbs = [
            { label: 'breadcrumb_pools', itemClass: 'active' }
        ];

        // Build metadata for displaying a list of pools
        $scope.pools = buildTable('ID', [
            { id: 'ID', name: 'pools_tbl_id'},
            { id: 'CoreCapacity', name: 'core_capacity'},
            { id: 'MemoryCapacity', name: 'memory_usage'},
            { id: 'CreatedAt', name: 'pools_tbl_created_at'},
            { id: 'UpdatedAt', name: 'updated_at'},
            { id: 'Actions', name: 'pools_tbl_actions'}
        ]);

        $scope.click_pool = function(id) {
            $location.path('/pools/' + id);
        };

        // Function to remove a pool
        $scope.clickRemovePool = function(poolID) {
            $modalService.create({
                template: $translate.instant("confirm_remove_pool") + "<strong>"+ poolID +"</strong>",
                model: $scope,
                title: "remove_pool",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "remove_pool",
                        classes: "btn-danger",
                        action: function(){
                            resourcesFactory.remove_pool(poolID, function(data) {
                                refreshPools($scope, resourcesFactory, false);
                            });
                            this.close();
                        }
                    }
                ]
            });
        };

        // Function for opening add pool modal
        $scope.modalAddPool = function() {
            $scope.newPool = {};
            $modalService.create({
                templateUrl: "add-pool.html",
                model: $scope,
                title: "add_pool",
                actions: [
                    {
                        role: "cancel",
                        action: function(){
                            $scope.newPool = {};
                            this.close();
                        }
                    },{
                        role: "ok",
                        label: "add_pool",
                        action: function(){
                            if(this.validate()){
                                // disable ok button, and store the re-enable function
                                var enableSubmit = this.disableSubmitButton();

                                $scope.add_pool()
                                    .success(function(data, status){
                                        $notification.create("Added new Pool", data.Detail).success();
                                        this.close();
                                    }.bind(this))
                                    .error(function(data, status){
                                        this.createNotification("Adding pool failed", data.Detail).error();
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ]
            });
        };

        // Function for adding new pools - through modal
        $scope.add_pool = function() {
            return resourcesFactory.add_pool($scope.newPool)
                .success(function(data){
                    refreshPools($scope, resourcesFactory, false);
                    // Reset for another add
                    $scope.newPool = {};
                });
        };

        // Ensure we have a list of pools
        refreshPools($scope, resourcesFactory, false);
    }]);
})();
