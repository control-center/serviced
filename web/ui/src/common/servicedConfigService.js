/* servicedConfigService.js
 * get UI config values from serviced
 */
(function(){
    "use strict";

    var resourcesFactory, $q;

    class ServicedConfig {
        constructor(){

        }

        update(){
            let d = $q.defer();

            resourcesFactory.getUIConfig()
                // TODO - errors
                .then(response => {
                    d.resolve(response);
                },
                err => {
                    d.reject(err);
                });

            this._d = d.promise;
            return this._d;
        }

        getConfig(){
            if(!this._d){
                this.update();
            }
            return this._d;
        }
    }

    var servicedConfig = new ServicedConfig();

    angular.module("servicedConfig", [])
    .factory("servicedConfig", ["$q", "resourcesFactory",
    function(_$q, _resourcesFactory){
        resourcesFactory = _resourcesFactory;
        $q = _$q;
        servicedConfig.update();
        return servicedConfig;
    }]);

})();
