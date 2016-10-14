(function() {
    'use strict';
    // access permissions that apply to pools
    controlplane.constant("POOL_PERMISSIONS", [
        {
            label: "Admin",
            description: "admin_permissions",
            position: 1 << 0
        },{
            label: "DFS",
            description: "dfs_permissions",
            position: 1 << 1
        }
    ]);

})();
