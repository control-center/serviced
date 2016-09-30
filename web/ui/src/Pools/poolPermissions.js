(function() {
    'use strict';
    // access permissions that apply to pools
    controlplane.constant("POOL_PERMISSIONS", [
        {
            label: "Admin",
            description: "Allow hosts to start, stop and restart services",
            position: 1 << 0
        },{
            label: "DFS",
            description: "Allow hosts to read and write to the DFS",
            position: 1 << 1
        }
    ]);

})();
