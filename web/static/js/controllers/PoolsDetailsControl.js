function PoolDetailsControl($scope, $routeParams, $location, resourcesService, authService, $modalService, $translate) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "pooldetails";
    $scope.params = $routeParams;

    $scope.breadcrumbs = [
        { label: 'breadcrumb_pools', url: '#/pools' }
    ];

    // Build metadata for displaying a pool's virtual ips
    $scope.virtual_ip_addresses = buildTable('IP', [
        { id: 'IP', name: 'pool_tbl_virtual_ip_address_ip'},
        { id: 'Netmask', name: 'pool_tbl_virtual_ip_address_netmask'},
        { id: 'BindInterface', name: 'pool_tbl_virtual_ip_address_bind_interface'},
        { id: 'Actions', name: 'pool_tbl_virtual_ip_address_action'}
    ]);

    //
    // Scope methods
    //

    // Pool view action - delete
    $scope.clickRemoveVirtualIp = function(ip) {
        console.log( "Removing pool's virtual ip address: ", ip);

        $modalService.create({
            template: $translate.instant("confirm_remove_virtual_ip") + " <strong>"+ ip.IP +"</strong>",
            model: $scope,
            title: "remove_virtual_ip",
            actions: [
                {
                    role: "cancel"
                },{
                    role: "ok",
                    label: "remove_virtual_ip",
                    classes: "btn-danger",
                    action: function(){
                        resourcesService.remove_pool_virtual_ip(ip.PoolID, ip.IP, function() {
                            refreshPools($scope, resourcesService, false);
                        });
                        // NOTE: should wait for success before closing
                        this.close();
                    }
                }
            ]
        });
        
        
    };

    // Add Virtual Ip Modal - Add button action
    $scope.AddVirtualIp = function(pool) {
        var ip = $scope.pools.add_virtual_ip;
        resourcesService.add_pool_virtual_ip(ip.PoolID, ip.IP, ip.Netmask, ip.BindInterface, function() {
            $scope.pools.add_virtual_ip = {};
            refreshPools($scope, resourcesService, false);
        });
    };

    // Open the virtual ip modal
    $scope.modalAddVirtualIp = function(pool) {
        $scope.pools.add_virtual_ip = {'PoolID': pool.ID, 'IP':"", 'Netmask':"", 'BindInterface':""};
        $modalService.create({
            templateUrl: "pool-add-virtualip.html",
            model: $scope,
            title: "add_virtual_ip",
            actions: [
                {
                    role: "cancel"
                },{
                    role: "ok",
                    label: "add_virtual_ip",
                    action: function(){
                        if(this.validate()){
                            $scope.AddVirtualIp(pool);
                            // NOTE: should wait for success before closing
                            this.close();
                        }
                    }
                }
            ]
        });
    };

    // Ensure we have a list of pools
    refreshPools($scope, resourcesService, true, function() {
        if ($scope.pools.current) {
            $scope.breadcrumbs.push({label: $scope.pools.current.ID, itemClass: 'active'});
        }
    });
}
