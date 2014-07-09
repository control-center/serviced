function HostsControl($scope, $routeParams, $location, $filter, $timeout, resourcesService, authService, $modalService){
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "hosts";
    $scope.params = $routeParams;

    $scope.breadcrumbs = [
        { label: 'breadcrumb_hosts', itemClass: 'active' }
    ];

    $scope.itemClass = itemClass;
    $scope.indent = indentClass;
    $scope.newHost = {};

    $scope.modalAddHost = function() {
        // $('#addHost').modal('show');
        $modalService.create({
            templateUrl: "add-host.html",
            model: $scope,
            title: "title_add_host",
            actions: [
                {
                    role: "cancel",
                    action: function(){
                        $scope.newHost = {};
                        this.close();
                    }
                },{
                    role: "ok",
                    label: "Add Host",
                    action: function(){
                        if(this.validate()){
                            $scope.add_host();
                            // NOTE: should wait for success before closing
                            this.close();
                        }
                    }
                }
            ]
        });
    };
    
    $scope.add_host = function() {
        resourcesService.add_host($scope.newHost, function(data) {
            // After adding, refresh our list
            refreshHosts($scope, resourcesService, false, hostCallback);
        });
        // Reset for another add
        $scope.newHost = {
            poolID: $scope.params.poolID
        };
    };

    $scope.clickHost = function(hostId) {
        $location.path('/hosts/' + hostId);
    };

    $scope.clickPool = function(poolID) {
        $location.path('/pools/' + poolID);
    };

    $scope.dropped = [];

    $scope.filterHosts = function() {
        if (!$scope.hosts.filtered) {
            $scope.hosts.filtered = [];
        }
        // Run ordering filter, built in
        var ordered = $filter('orderBy')($scope.hosts.all, $scope.hosts.sort);
        // Run search filter, built in
        var filtered = $filter('filter')(ordered, $scope.hosts.search);
        // Run filter for pool and child pools, custom
        var treeFiltered = $filter('treeFilter')(filtered, 'poolID', $scope.subPools);

        // As a side effect, save number of hosts before paging
        if (treeFiltered) {
            $scope.hosts.filteredCount = treeFiltered.length;
        } else {
            $scope.hosts.filteredCount = 0;
        }
        var page = $scope.hosts.page? $scope.hosts.page : 1;
        var pageSize = $scope.hosts.pageSize? $scope.hosts.pageSize : 5;
        var itemsToTake = page * pageSize;
        $scope.hosts.filteredCountLimit = itemsToTake;
        if (treeFiltered) {
            $scope.hosts.filtered = treeFiltered.splice(0, itemsToTake);
        }
        return $scope.hosts.filtered;
    };

    $scope.loadMore = function() {
        if ($scope.hosts.filteredCount && $scope.hosts.filteredCountLimit &&
            $scope.hosts.filteredCountLimit < $scope.hosts.filteredCount) {
            $scope.hosts.page += 1;
            $scope.filterHosts();
            return true;
        }

        return false;
    };

    // Build metadata for displaying a list of hosts
    $scope.hosts = buildTable('Name', [
        { id: 'Name', name: 'Name'},
        { id: 'fullPath', name: 'Assigned Resource Pool'},
    ]);

    var hostCallback = function() {
        $scope.hosts.page = 1;
        $scope.hosts.pageSize = 10;
        $scope.filterHosts();
    };

    // Ensure we have a list of pools
    refreshPools($scope, resourcesService, false);

    // Also ensure we have a list of hosts
    refreshHosts($scope, resourcesService, false, hostCallback);
}