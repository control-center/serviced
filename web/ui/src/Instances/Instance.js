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

            // the instance model data comes in with health and
            // memory stats, so use that to do an initial instace
            // status update
            this.updateState({
                HealthStatus: instance.HealthStatus,
                MemoryUsage: instance.MemoryUsage
            });
        }

        resourcesGood() {
            return this.resources.RAMLast < this.resources.RAMCommitment;
        }

        // update fast-moving instance state
        updateState(status) {
            this.resources.RAMAverage = Math.max(0, status.MemoryUsage.Avg);
            this.resources.RAMLast = Math.max(0, status.MemoryUsage.Cur);
            this.resources.RAMMax = Math.max(0, status.MemoryUsage.Max);
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
