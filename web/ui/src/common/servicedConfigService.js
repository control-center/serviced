/* servicedConfigService.js
 * get UI config values from serviced
 */
(function(){
    "use strict";

    var $http, $q, $cookies, log;

    class ServicedConfig {
        constructor(){
            // TODO - gather all defaults here
            this._config = {
                PollFrequency: 10
            };
        }

        _update(){
            let d = $q.defer();

            // grab cookie config values
            this._config.ZUsername = $cookies.get("ZUsername");
            this._config.ZCPToken = $cookies.get("ZCPToken");
            this._config.Language = $cookies.get("Language");
            this._config.autoRunWizardHasRun = $cookies.get("autoRunWizardHasRun");

            // get config values from serviced
            // NOTE - using $http directly instead of resource service
            // because resource service automatically redirects on
            // unauthorized, and we want to just try again here
            $http.get("/config")
                .then(response => {
                    // TODO - handle error response
                    this._config = angular.merge(this._config, response.data);
                    d.resolve(this._config);
                    log.info(this._config);
                },
                err => {
                    let errMessage = err.statusText;
                    if(err.data && err.data.Detail){
                        errMessage = err.data.Detail;
                    }
                    log.error("failed to load serviced config with error:", "'"+ errMessage +"'.", "using default values");
                    log.info(this._config);
                    d.resolve(this._config);
                    // this allows the next call to try again
                    this._d = undefined;
                });

            this._d = d.promise;
            return this._d;
        }

        // fetches config if not already fetched and
        // waits till all configs have been gathered
        // before fulfilling a promise.
        update(){
            if(!this._d){
                this._update();
            }
            return this._d;
        }

        // gets a config value, but does not attempt
        // to fetch any missing config values
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
    .factory("servicedConfig", ["$q", "$http", "$cookies", "log",
    function(_$q, _$http, _$cookies, _log){
        $http = _$http;
        $q = _$q;
        $cookies = _$cookies;
        log = _log;
        servicedConfig._update();
        return servicedConfig;
    }]);

})();
