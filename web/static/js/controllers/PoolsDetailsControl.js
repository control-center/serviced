function PoolDetailsControl($scope, $routeParams, $location, resourcesService, authService, statsService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "pooldetails";
    $scope.params = $routeParams;

    $scope.breadcrumbs = [
        { label: 'breadcrumb_pools', itemClass: 'active'}
    ];

    // Build metadata for displaying a pool's virtual ips
    $scope.virtual_ip_addresses = buildTable('Address', [
        { id: 'Address', name: 'pool_tbl_virtual_ip_address'},
        { id: 'Actions', name: 'pool_tbl_virtual_ip_address_action'}
    ]);

    // Scope methods
    $scope.clickRemoveVirtualIp = function(pool, ip) {
        console.log( "Removing pool's virtual ip address: ", pool, ip);
        resourcesService.remove_pool_virtual_ip(pool.ID, ip, function() {
            refreshPools($scope, resourcesService, false);
        });
    };

    $scope.modalAddVirtualIp = function(pool) {
        $scope.pools.add_virtual_ip = {'id': pool.ID, 'ip':""};
        $('#poolAddVirtualIp').modal('show');
    };

    $scope.AddVirtualIp = function(pool) {
        var poolID = $scope.pools.add_virtual_ip.id;
        var ip = $scope.pools.add_virtual_ip.ip;
        resourcesService.add_pool_virtual_ip(poolID, ip, function() {
            $scope.pools.add_virtual_ip.ip = "";
        });
        $('#poolAddVirtualIp').modal('hide');
    };

    $scope.CancelAddVirtualIp = function(pool) {
        $scope.pools.add_virtual_ip = null;
        $('#poolAddVirtualIp').modal('hide');
    };

    // Ensure we have a list of pools
    refreshPools($scope, resourcesService, true, function() {
        if ($scope.pools.current) {
            $scope.breadcrumbs.push({label: $scope.pools.current.Id, itemClass: 'active'});
        }
    });
}