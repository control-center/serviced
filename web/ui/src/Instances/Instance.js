(function () {
    'use strict';

    // share angular services outside of angular context
    let utils;

    controlplane.factory('Instance', InstanceFactory);

    function buildStateId(hostid, serviceid, instanceid) {
        return `${hostid}-${serviceid}-${instanceid}`;
    }

    class Instance {

        constructor(instance) {
            this.model = Object.freeze(instance);
            this.id = buildStateId(this.model.HostID, this.model.ServiceID, this.model.InstanceID);

            this.resources = {
                RAMCommitment: utils.parseEngineeringNotation(instance.RAMCommitment)
            };
            console.log(`Health constructor`);

            this.updateState({
                HealthStatus: instance.HealthStatus,
                MemoryUsage: instance.MemoryUsage
            });
        }

        resourcesGood() {
            return this.resources.RAMLast < this.resources.RAMCommitment;
        }

        updateState(status) {
            this.resources.RAMAverage = Math.max(0, status.MemoryUsage.Avg);
            this.resources.RAMLast = Math.max(0, status.MemoryUsage.Cur);
            this.resources.RAMMax = Math.max(0, status.MemoryUsage.Max);
            console.log(`Health setting Health property from status object`);
            this.healthChecks = status.HealthStatus;
        }

    }


    InstanceFactory.$inject = ['miscUtils'];

    function InstanceFactory(_utils) {

        // api access via angular context
        utils = _utils;

        return Instance;
    }
})();