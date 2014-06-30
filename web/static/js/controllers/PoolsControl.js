function PoolsControl($scope, $routeParams, $location, $filter, $timeout, resourcesService, authService) {
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
        { id: 'Priority', name: 'pools_tbl_priority'},
        { id: 'CoreCapacity', name: 'pools_tbl_core_capacity'},
        { id: 'MemoryCapacity', name: 'pools_tbl_memory_capacity'},
        { id: 'MemoryCommitment', name: 'pools_tbl_memory_commitment'},
        { id: 'CreatedAt', name: 'pools_tbl_created_at'},
        { id: 'UpdatedAt', name: 'pools_tbl_updated_at'},
        { id: 'Actions', name: 'pools_tbl_actions'}
    ])

    $scope.click_pool = function(id) {
        $location.path('/pools/' + id);
    };

    // Function to remove a pool
    $scope.clickRemovePool = function(poolID) {
        console.log( "Click Remove pool w/id: ", poolID);
        resourcesService.remove_pool(poolID, function(data) {
            refreshPools($scope, resourcesService, false);
        });
    };

    // Function for opening add pool modal
    $scope.modalAddPool = function() {
        $scope.newPool = {};
        $('#addPool').modal('show');
    };

    // Function for adding new pools - through modal
    $scope.add_pool = function() {
        console.log('Adding pool %s as child of pool %s', $scope.newPool.ID, $scope.params.poolID);
        resourcesService.add_pool($scope.newPool, function(data) {
            refreshPools($scope, resourcesService, false);
        });
        // Reset for another add
        $scope.newPool = {};
    };

    // Ensure we have a list of pools
    refreshPools($scope, resourcesService, true);
}
