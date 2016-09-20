(function() {
    'use strict';

    // access permissions that apply to pools
    // TODO - translations
    controlplane.constant("POOL_PERMISSIONS", [
        {
            field: "DFSAccess",
            label: "DFS",
            description: "Allow hosts to read and write to the DFS"
        },{
            field: "AdminAccess",
            label: "Admin",
            description: "Allow hosts to start, stop and restart services"
        }
    ]);

})();
