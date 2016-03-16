/* miscUtils.js
 * miscellaneous utils and stuff that
 * doesn't quite fit in elsewhere
 */
(function(){
    "use strict";

    angular.module("miscUtils", [])
    .factory("miscUtils", [ "$parse",
    function($parse){

        //polyfill endsWith so phantomjs won't complain :/
        if (!String.prototype.endsWith) {
          String.prototype.endsWith = function(searchString, position) {
              var subjectString = this.toString();
              if (typeof position !== 'number' || !isFinite(position) || Math.floor(position) !== position || position > subjectString.length) {
                position = subjectString.length;
              }
              position -= searchString.length;
              var lastIndex = subjectString.indexOf(searchString, position);
              return lastIndex !== -1 && lastIndex === position;
          };
        }

        var utils = {

            // TODO - use angular $location object to make this testable
            unauthorized: function() {
                console.error('You don\'t appear to be logged in.');
                // show the login page and then refresh so we lose any incorrect state. CC-279
                window.location.href = "/#/login";
                window.location.reload();
            },

            indentClass: function(depth) {
                return 'indent' + (depth -1);
            },

            downloadFile: function(url){
                window.location = url;
            },

            getModeFromFilename: function(filename){
                var re = /(?:\.([^.]+))?$/;
                var ext = re.exec(filename)[1];
                var mode;
                switch(ext) {
                    case "conf":
                        mode="properties";
                        break;
                    case "xml":
                        mode = "xml";
                        break;
                    case "yaml":
                        mode = "yaml";
                        break;
                    case "txt":
                        mode = "plain";
                        break;
                        case "json":
                        mode = "javascript";
                        break;
                    default:
                        mode = "shell";
                        break;
                }

                return mode;
            },

            updateLanguage: function($scope, $cookies, $translate){
                var ln = 'en_US';
                if ($cookies.Language === undefined) {

                } else {
                    ln = $cookies.Language;
                }
                if ($scope.user) {
                    $scope.user.language = ln;
                }
                $translate.use(ln);
            },

            capitalizeFirst: function(str){
                return str.slice(0,1).toUpperCase() + str.slice(1);
            },

            // call fn b after fn a
            after: function(a, b, context){
                return function(){
                    var results;
                    results = a.apply(context, arguments);
                    // TODO - send results to b?
                    b.call(context);
                    return results;
                };
            },

            mapToArr: function(data) {
                var arr = [];
                for (var key in data) {
                    arr.push(data[key]);
                }
                return arr;
            },


            // cache function results based on hash function.
            // NOTE: unlike regular memoize, the caching is entirely
            // based on hash function, not on arguments
            memoize: function(fn, hash){
                var cache = {};
                return function(){
                    var key = hash.apply(this, arguments),
                        val;

                    // if value isnt cached, evaluate and cache
                    if(!(key in cache)){
                        val = fn.apply(this, arguments);
                        cache[key] = val;
                    } else {
                        val = cache[key];
                    }

                    return val;
                };
            },

            needsHostAlias: function(host){
                // check is location.hostname is an IP
                var re = /\b(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b/;
                return re.test(host) || host === "localhost";
            },

            parseEngineeringNotation: function(str) {
                // Converts nK/k, nM/m, nG/g, nT/t to a number. e.g. 1K returns 1024.
                if (str === "" || str === undefined) {
                    return 0;
                }

                // if this is already a regular, boring, ol number
                if(isFinite(+str)){
                    return +str;
                }

                var prefix = parseFloat(str);
                var suffix = str.slice(prefix.toString().length);
                switch(suffix) {
                    case "K":
                    case "k":
                        prefix *= (1 << 10);
                        break;
                    case "M":
                    case "m":
                        prefix *= (1 << 20);
                        break;
                    case "G":
                    case "g":
                        prefix *= (1 << 30);
                        break;
                    case "T":
                    case "t":
                        prefix *= (1 << 40);
                        break;
                    default:
                        prefix = undefined;
                        break;
                }
                return prefix;
            },

            // returns a function that will parse the
            // expression `attr` on scope object $scope
            // and return that value
            propGetter: function($scope, attr){
                var getter = $parse(attr);
                return function(){
                    return getter($scope);
                };
            },

            // TODO - make services that should not count configurable
            // eg: started, stopped, manual, etc
            countTheKids: function(parentService, filterFunction=() => true){
                var children = parentService.children || [],
                    childCount = 0;

                // count number of descendent services that will start
                childCount = children.reduce(function countTheKids(acc, service){
                    // if a service is not set to manual launch and
                    // has a startup command, it probably should count
                    var shouldCount = service.model.Launch !== "manual" &&
                        service.model.Startup;

                    // if shouldCount and the filter function returns
                    // true, this definitely counts
                    if(shouldCount && filterFunction(service)){
                        acc++;
                    }

                    // if the service has children, check em
                    if(service.children){
                        return service.children.reduce(countTheKids, acc);
                    }
                }, 0);

                return childCount;
            },

            validateHostName: function(hostStr, $translate){
                if (hostStr === undefined || hostStr === '') {
                    return $translate.instant("content_wizard_invalid_host");
                }

                return null;
            },

            validatePortNumber: function(port, $translate){
                if (port === undefined || port === '') {
                    return $translate.instant("port_number_invalid");
                }
                if(+port < 1 || +port > 65535){
                    return $translate.instant("port_number_invalid_range");
                }

                return null;
            },

            validateRAMLimit: function(limitStr, max=Infinity){

                if (limitStr === undefined || limitStr === '') {
                    return null;
                }

                var isPercent = (limitStr.endsWith("%"));
                var isEngineeringNotation = /.*[KkMmGgTt]$/.test(limitStr);

                if (!isPercent && !isEngineeringNotation) {
                    return "Invalid RAM Limit value, must specify % or unit of K, M, G, or T";
                }

                // if this is a percent, ensure its between 1 and 100
                if(isPercent){
                    let val = +limitStr.slice(0, -1);
                    if(val > 100){
                        return "RAM Limit cannot exceed 100%";
                    }
                    if(val <= 0){
                        return "RAM Limit must be at least 1%";
                    }

                // if this is a byte value, ensure its less than host memory
                } else {
                    let val = utils.parseEngineeringNotation(limitStr);
                    if(isNaN(val) || val === undefined){
                        return "Invalid RAM Limit value";
                    }
                    if(val > max){
                        return "RAM Limit exceeds available host memory";
                    }
                    if(val <= 0){
                        return "RAM Limit must be at least 1";
                    }

                }
                return null;
            }
       };

        return utils;
    }]);
})();
