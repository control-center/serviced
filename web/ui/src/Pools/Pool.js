(function () {
    'use strict';

    // share angular services outside of angular context
    let POOL_PERMISSIONS, $q, resourcesFactory, Host;

    controlplane.factory('Pool', PoolFactory);

    // Pool object constructor takes a pool object (backend pool object)
    // and wraps it with extra functionality and info
    class Pool{
    
        constructor(pool){
            this.id = pool.ID;
            this.model = Object.freeze(pool);
            this.hosts = [];
            this.calculatePermissions();
            this.touch();
        }

        // mark services updated to trigger render via $watch
        touch() {
            this.lastUpdate = new Date().getTime();
        }

        calculatePermissions(){
            // build a list of permissions
            // this pool has
            // NOTE: permissions include description
            // and friendly label for the UI to display
            let val = this.model.Permissions;
            this.permissions = [];
            POOL_PERMISSIONS.forEach(perm => {
                if((val & perm.position) !== 0){
                    this.permissions.push(perm);
                }
            });
        }

        fetchHosts(force){
            let deferred = $q.defer();
            if (this.hosts && !force) {
                deferred.resolve();
            }
            // TODO - this is actually a v2 endpoint
            // and should be on resourcesFactory.v2
            resourcesFactory.getPoolHosts(this.id)
                .then(data => {
                    this.hosts = data.map(h => new Host(h));
                    this.touch();
                    deferred.resolve();
                },
                error => {
                    console.warn(error);
                    deferred.reject();
                });
            return deferred.promise;
        }
    }


    PoolFactory.$inject = ['POOL_PERMISSIONS', '$q', 'resourcesFactory', 'Host'];

    function PoolFactory(_POOL_PERMISSIONS, _$q, _resourcesFactory, _Host) {

        // api access via angular context
        POOL_PERMISSIONS = _POOL_PERMISSIONS;
        $q = _$q;
        resourcesFactory = _resourcesFactory;
        Host = _Host;

        return Pool;
    }
})();
