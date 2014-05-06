function HostsControl($scope, $routeParams, $location, $filter, $timeout, resourcesService, authService){
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "hosts";
    $scope.params = $routeParams;

    $scope.breadcrumbs = [
        { label: 'breadcrumb_hosts', itemClass: 'active' }
    ];

    $scope.toggleCollapsed = function(toggled) {
        toggled.collapsed = !toggled.collapsed;
        if (toggled.children === undefined) {
            return;
        }
        toggled.icon = toggled.collapsed? POOL_ICON_CLOSED : POOL_ICON_OPEN;
        for (var i=0; i < toggled.children.length; i++) {
            toggleCollapse(toggled.children[i], toggled.collapsed);
        }
    };
    $scope.itemClass = itemClass;
    $scope.indent = indentClass;
    $scope.newPool = {};
    $scope.newHost = {};

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

    $scope.addSubpool = function(poolID) {
        $scope.newPool.ParentId = poolID;
        $('#addPool').modal('show');
    };
    $scope.delSubpool = function(poolID) {
        resourcesService.remove_pool(poolID, function(data) {
            refreshPools($scope, resourcesService, false, function(){ removePool($scope, poolID) });
        });
    };

    $scope.modalAddHost = function() {
        $('#addHost').modal('show');
    };

    // Build metadata for displaying a list of pools
    $scope.pools = buildTable('Id', [
        { id: 'Id', name: 'Id'},
        { id: 'ParentId', name: 'Parent Id'},
        { id: 'Priority', name: 'Priority'}
    ])

    var clearLastStyle = function() {
        var lastPool = $scope.pools.mapped[$scope.selectedPool];
        if (lastPool) {
            lastPool.current = false;
        }
    };

    $scope.clearSelectedPool = function() {
        clearLastStyle();
        $scope.selectedPool = null;
        $scope.subPools = null;
        hostCallback();
    };

    $scope.clickHost = function(hostId) {
        $location.path('/hosts/' + hostId);
    };

    $scope.clickPool = function(poolID) {
        var topPool = $scope.pools.mapped[poolID];
        if (!topPool || $scope.selectedPool === poolID) {
            $scope.clearSelectedPool();
            return;
        }
        clearLastStyle();
        topPool.current = true;

        var allowed = {};
        addChildren(allowed, topPool);
        $scope.subPools = allowed;
        $scope.selectedPool = poolID;
        hostCallback();
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

    $scope.dropIt = function(event, ui) {
        var poolID = $(event.target).attr('data-pool-id');
        var pool = $scope.pools.mapped[poolID];
        var host = $scope.dropped[0];

        if (poolID === host.poolID) {
            // Nothing changed. Don't bother showing the dialog.
            return;
        }

        $scope.move = {
            host: host,
            newpool: poolID
        };
        $scope.dropped = [];
        $('#confirmMove').modal('show');
    };

    $scope.confirmMove = function() {
        console.log('Reassigning %s to %s', $scope.move.host.Name, $scope.move.newpool);
        var modifiedHost = $.extend({}, $scope.move.host);
        modifiedHost.poolID = $scope.move.newpool;
        resourcesService.update_host(modifiedHost.ID, modifiedHost, function() {
            refreshHosts($scope, resourcesService, false, hostCallback);
        });
    };

    // Function for adding new pools
    $scope.add_pool = function() {
        console.log('Adding pool %s as child of pool %s', $scope.newPool.ID, $scope.params.poolID);
        resourcesService.add_pool($scope.newPool, function(data) {
            // After adding, refresh our list
            refreshPools($scope, resourcesService, false);
        });
        // Reset for another add
        $scope.newPool = {};
    };

    // Function for removing the current pool
    $scope.remove_pool = function() {
        console.log('Removing pool %s', $scope.params.poolID);
        resourcesService.remove_pool($scope.params.poolID, function(data) {
            refreshPools($scope, resourcesService, false);
        });
    };

    // Build metadata for displaying a list of hosts
    $scope.hosts = buildTable('Name', [
        { id: 'Name', name: 'Name'},
        { id: 'fullPath', name: 'Assigned Resource Pool'},
    ]);

    $scope.clickMenu = function(index) {
        $('#pool_menu_' + index).addClass('tempvis');
        setTimeout(function() {
            $('#pool_menu_' + index).removeClass('tempvis');
        }, 600);
    };

    var hostCallback = function() {
        $scope.hosts.page = 1;
        $scope.hosts.pageSize = 10;
        $scope.filterHosts();
        $timeout($scope.hosts.scroll, 100);
    };

    // Ensure we have a list of pools
    refreshPools($scope, resourcesService, false);
    // Also ensure we have a list of hosts
    refreshHosts($scope, resourcesService, false, hostCallback);
}