function HostsMapControl($scope, $routeParams, $location, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "hostsmap";
    $scope.params = $routeParams;
    $scope.indent = indentClass;
    $scope.breadcrumbs = [
        { label: 'breadcrumb_hosts', url: '#/hosts' },
        { label: 'breadcrumb_hosts_map', itemClass: 'active' }
    ];

    var width = 857;
    var height = 567;

    var cpuCores = function(h) {
        return h.Cores;
    };
    var memoryCapacity = function(h) {
        return h.Memory;
    };
    var poolBgColor = function(p) {
        return p.isHost? null : color(p.Id);
    };
    var hostText = function(h) {
        return h.isHost? h.Name : null;
    };

    var color = d3.scale.category20c();
    var treemap = d3.layout.treemap()
        .size([width, height])
        .value(memoryCapacity);

    var position = function() {
        this.style("left", function(d) { return d.x + "px"; })
            .style("top", function(d) { return d.y + "px"; })
            .style("width", function(d) { return Math.max(0, d.dx - 1) + "px"; })
            .style("height", function(d) { return Math.max(0, d.dy - 1) + "px"; });
    };

    $scope.selectionButtonClass = function(id) {
        var cls = 'btn btn-link nav-link';
        if ($scope.treemapSelection === id) {
            cls += ' active';
        }
        return cls;
    };

    $scope.selectByMemory = function() {
        $scope.treemapSelection = 'memory';
        selectNewValue(memoryCapacity);
    };
    $scope.selectByCores = function() {
        $scope.treemapSelection = 'cpu';
        selectNewValue(cpuCores);
    };

    var selectNewValue = function(valFunc) {
        var node = d3.select("#hostmap").
            selectAll(".node").
            data(treemap.value(valFunc).nodes);
        node.enter().
            append("div").
            attr("class", "node");
        node.transition().duration(1000).
            call(position).
            style("background", poolBgColor).
            text(hostText);
        node.exit().remove();
    };

    var selectNewRoot = function(newroot) {
        console.log('Selected %s', newroot.Id);
        var node = d3.select("#hostmap").
            datum(newroot).
            selectAll(".node").
            data(treemap.nodes);

        node.enter().
            append("div").
            attr("class", "node");

        node.transition().duration(1000).
            call(position).
            style("background", poolBgColor).
            text(hostText);
        node.exit().remove();
    };

    var hostsAddedToPools = false;
    var wait = { pools: false, hosts: false };
    var addHostsToPools = function() {
        if (!wait.pools || !wait.hosts) {
            return;
        }
        if (hostsAddedToPools) {
            console.log('Already built');
            return;
        }

        console.log('Preparing tree map');
        hostsAddedToPools = true;
        for(var key in $scope.hosts.mapped) {
            var host = $scope.hosts.mapped[key];
            var pool = $scope.pools.mapped[host.PoolID];
            if (pool.children === undefined) {
                pool.children = [];
            }
            pool.children.push(host);
            host.isHost = true;
        }
        var root = { Id: 'All Resource Pools', children: $scope.pools.tree };
        selectNewRoot(root);
    };
    $scope.treemapSelection = 'memory';
    // Also ensure we have a list of hosts
    refreshPools($scope, resourcesService, false, function() {
        wait.pools = true;
        addHostsToPools();
    });
    refreshHosts($scope, resourcesService, false, function() {
        wait.hosts = true;
        addHostsToPools();
    });
}
