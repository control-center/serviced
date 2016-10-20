(function () {
    'use strict';

    let $serviceHealth, resourcesFactory;

    controlplane.factory('InternalService', InternalServiceFactory);
  
    class InternalServiceInstance {

        constructor(data) {
            this.id = data.InstanceID;
            this.name = data.ServiceName;
            this.model = Object.freeze(data);

            this.touch();
        }
        
        touch() {
            this.lastUpdate = new Date().getTime();
        }

        updateStatus(status) {
            this.healthChecks = status.HealthStatus;
            this.touch();
        }
    }

    class InternalService {

        constructor(data){
            this.id = data.ID;
            this.name = data.Name;
            this.model = Object.freeze(data);

            this.touch();
        }

        touch() {
            this.lastUpdate = new Date().getTime();
        }

        fetchInstances() {
            return resourcesFactory.v2.getInternalServiceInstances(this.id)
                .then(data => this.instances = data.map(i => new InternalServiceInstance(i)));
        }

        updateStatus(status) {
            this.desiredState = status.DesiredState;
            
            let statusMap = status.Status.reduce((map, s) => {
                map[s.InstanceID] = s;
                return map;
            }, {});
            
            this.instances.forEach(i => {
                let s = statusMap[i.id];
                if (s) {
                    i.updateStatus(s);
                } else {
                    console.log(`Could not find status for instance ${i.id}`);
                }
            });

            $serviceHealth.update({ [this.id]: this });
            this.status = $serviceHealth.get(this.id);
            this.touch();
        }
    }
    
    InternalServiceFactory.$inject = ['$serviceHealth', 'resourcesFactory'];
    function InternalServiceFactory(_$serviceHealth, _resourcesFactory) {

        $serviceHealth = _$serviceHealth;
        resourcesFactory = _resourcesFactory;

        return InternalService;
    }

})();
