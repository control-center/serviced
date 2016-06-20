/* servicedConfigService.js
 * get UI config values from serviced
 */
(function(){
    "use strict";

    var resourcesFactory, $q, $cookies, log;

    class ServicedConfig {
        constructor(){
            this._config = {};
        }

        _update(){
            let d = $q.defer();

            // grab cookie config values
            this._config.ZUsername = $cookies.get("ZUsername");
            this._config.ZCPToken = $cookies.get("ZCPToken");
            this._config.Language = $cookies.get("Language");
            this._config.autoRunWizardHasRun = $cookies.get("autoRunWizardHasRun");

            // get config values from serviced
            resourcesFactory.getUIConfig()
                // TODO - errors
                .then(response => {
                    this._config = angular.merge(this._config, response);
                    d.resolve(this._config);
                    log.info(this._config);
                },
                err => {
                    d.reject(err);
                });

            this._d = d.promise;
            return this._d;
        }

        getConfig(){
            if(!this._d){
                this._update();
            }
            return this._d;
        }

        get(key){
            let val = this._config[key];
            if(val === "true"){
                val = true;
            } else if(val === "false"){
                val = false;
            } else if(val === "undefined"){
                val = undefined;
            }
            return val;
        }

        // NOTE - this should only be used for cookie values
        set(key, val){
            // NOTE: cookies only handle strings
            if(val === true){
                val = "true";
            } else if(val === false){
                val = "false";
            } else if(val === undefined){
                val = "undefined";
            }
            // TODO - distinguish between values stored
            // in cookies or in serviced
            $cookies.put(key, val);
            this._config[key] = val;
        }
    }

    var servicedConfig = new ServicedConfig();

    angular.module("servicedConfig", [])
    .factory("servicedConfig", ["$q", "resourcesFactory", "$cookies", "log",
    function(_$q, _resourcesFactory, _$cookies, _log){
        resourcesFactory = _resourcesFactory;
        $q = _$q;
        $cookies = _$cookies;
        log = _log;
        servicedConfig._update();
        return servicedConfig;
    }]);

})();
