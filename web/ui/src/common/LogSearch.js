/* globals rison: true */
/* log.js
 * logging, pure of heart.
 */
(function(){
    "use strict";

    let $location;

    const KIBANA_PATH = "/api/controlplane/kibana";
    const DEFAULT_INDEX = "logstash-*";

    const DEFAULT_APPCONFIG = {
        columns: ["fields.type", "message"],
        filters: [],
        index: DEFAULT_INDEX,
        interval: "auto",
        query: {
            query_string: {
                analyze_wildcard: true,
                query: "fields.type:*"
            }
        },
        sort: ["@timestamp", "desc"],
        vis: {}
    };

    const DEFAULT_GLOBALCONFIG = {
        refreshInterval: {
            display: "Off",
            pause: false,
            value: 0
        },
        time: {
            from: "now-30d",
            mode: "quick",
            to: "now"
        }
    };

    function generateKibanaURL(baseURL, appConfig, globalConfig, index){
            return `
${baseURL}/app/kibana?#/discover?
_g=${rison.encode(globalConfig)}&
_a=${rison.encode(appConfig)}&
indexPattern=${index}&type=histogram`;
    }

    class LogSearch {
        constructor(){
            let port = $location.port() ? `:${$location.port()}` : ``,
                host = $location.host(),
                protocol = $location.protocol(),
                origin = `${protocol}://${host}${port}`;

            this.baseURL = `${origin}${KIBANA_PATH}`;
            console.log(this.baseURL);
        }

        // given an app config and global config, generate
        // a search url
        getURL(appConfig, globalConfig){
            return generateKibanaURL(this.baseURL, appConfig, globalConfig, DEFAULT_INDEX);
        }

        // generate the default search url
        getDefaultURL(){
            return generateKibanaURL(this.baseURL, DEFAULT_APPCONFIG, DEFAULT_GLOBALCONFIG, DEFAULT_INDEX);
        }

        // give other users some defaults
        // to make new requests from
        getDefaultAppConfig(){
            return angular.copy(DEFAULT_APPCONFIG);
        }
        getDefaultGlobalConfig(){
            return angular.copy(DEFAULT_GLOBALCONFIG);
        }
    }

    angular.module("LogSearch", [])
    .factory("LogSearch", ["$location", 
    function(_$location){
        $location = _$location;
        return new LogSearch();
    }]);
})();
