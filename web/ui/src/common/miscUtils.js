/* miscUtils.js
 * miscellaneous utils and stuff that
 * doesn't quite fit in elsewhere
 */
(function(){
    "use strict";

    var TIMEMULTIPLIER = {w: 6048e5, d: 864e5, h: 36e5, m: 6e4, s: 1e3,  ms: 1};
    var AUTH_IN_PROGRESS = false;

    angular.module("miscUtils", [])
    .factory("miscUtils", [ "$parse", "log", "angularAuth0",
    function($parse, log, angularAuth0){

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

        // polyfill find so IE doesnt complain :\
        if (!Array.prototype.find) {
          Object.defineProperty(Array.prototype, 'find', {
            value: function(predicate) {
             if (this === null || this === undefined) {
               throw new TypeError('Array.prototype.find called on null or undefined');
             }
             if (typeof predicate !== 'function') {
               throw new TypeError('predicate must be a function');
             }
             var list = Object(this);
             var length = list.length >>> 0;
             var thisArg = arguments[1];
             var value;

             for (var i = 0; i < length; i++) {
               value = list[i];
               if (predicate.call(thisArg, value, i, list)) {
                 return value;
               }
             }
             return undefined;
            }
          });
        }

        // fix for chrome 48 and up, as described here:
        // https://github.com/cpettitt/dagre-d3/issues/202
        SVGElement.prototype.getTransformToElement = SVGElement.prototype.getTransformToElement || function(elem) {
            return elem.getScreenCTM().inverse().multiply(this.getScreenCTM());
        };

        // creates a biset of specified size and
        // sets the value to to val. Also attaches
        // getter/setter functions angular can bind
        // to to toggle fields
        class NgBitset {
            constructor(size, val){
                this.val = val;
                this.size = size;

                // create angular getterSetters so that
                // this bitset can bind to checkboxes in the UI
                for(let i = 0; i < (1 << size); i = 1 << i){
                    this[i] = (val) => {
                        // if val, toggle the bit
                        // TODO - act based on val?
                        if(val !== undefined){
                            this.toggle(i);
                        }
                        return this.isSet(i);
                    };
                }
            }

            isSet(i){
                return (this.val & i) !== 0;
            }

            toggle(i){
                this.val = this.val ^ i;
            }

            // TODO - set/unset
        }


        var utils = {

            useAuth0: function() {
                if (window.Auth0Config.Auth0Scope && window.Auth0Config.Auth0Audience && window.Auth0Config.Auth0Domain && window.Auth0Config.Auth0ClientID) {
                    return true;
                }
                return false;
            },

            // TODO - use angular $location object to make this testable
            unauthorized: function() {
                log.error('You don\'t appear to be logged in.');

                if (utils.useAuth0()) {
                    if (AUTH_IN_PROGRESS) {
                        console.info("unauthorized(): auth in progress - do nothing.");
                    } else {
                        AUTH_IN_PROGRESS = true;
                        // first, see if we already have a login session
                        console.info("calling Auth0 checkSession()");
                        angularAuth0.checkSession({}, (err, result) => {
                            if (err) {
                                // no session or some other error - kick back to auth0 login
                                console.error("auth0 checkSession() returned an error: " + JSON.stringify(err));
                                AUTH_IN_PROGRESS = false;
                                console.info("calling Auth0 authorize()");
                                angularAuth0.authorize();
                            } else if (result && result.idToken && result.accessToken) {
                                // we got a session refresh from auth0. Update the token cookies and carry on.
                                window.sessionStorage.setItem("auth0AccessToken", result.accessToken);
                                window.sessionStorage.setItem("auth0IDToken", result.idToken);
                                AUTH_IN_PROGRESS = false;
                            } else {
                                // refresh worked, but didn't have tokens. Kick back to login screen.
                                AUTH_IN_PROGRESS = false;
                                window.location.href = "/#/login";
                                window.location.reload();
                            }
                        });
                    }
                } else {
                    // show the login page and then refresh so we lose any incorrect state. CC-279
                    window.location.href = "/#/login";
                    window.location.reload();
                }
            },

            indentClass: function(depth) {
                return 'indent' + (depth -1);
            },

            downloadFile: function(url){
                window.location = url;
            },

            // http://stackoverflow.com/a/18197341
			downloadText(filename, text) {
				var element = document.createElement('a');
				element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(text));
				element.setAttribute('download', filename);
				element.style.display = 'none';
				document.body.appendChild(element);
				element.click();
				document.body.removeChild(element);
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
                var ln = $cookies.get("Language") || "en_US";
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

            isIpAddress: function(addr) {
                var re = /\b(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b/;
                return re.test(addr);
            },

            needsHostAlias: function(host){
                // check is location.hostname is an IP
                return this.isIpAddress(host) || host === "localhost";
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

            humanizeDuration: function(msecs) {
                // converts millisecons to a time duration such as 4h45m20s

                if (msecs === 0) { return "0"; }

                var humanized = "";
                var ttoken = Object.keys(TIMEMULTIPLIER);
                
                for (var i=0; i<ttoken.length; i++) {
                    var d = Math.floor(msecs/TIMEMULTIPLIER[ttoken[i]]);
                    if (d) {
                        humanized += d.toString() + ttoken[i];
                        msecs -= (d * TIMEMULTIPLIER[ttoken[i]]);
                    }
                }
                // unused or duplicate tokens means bad input
                return humanized;
            },

            parseDuration: function(humanTime) {
                // converts time duration such as 4h45m20s into milliseconds
                // accepts weeks thru milliseconds as: w d h m s ms.
                // 23m45s8ms  1425008
                
                var human = humanTime.toString().toLowerCase().replace(/ /g,'');
                if (human === "0" || human === "") { return 0; }

                var badchars = human.match(/[^\da-z]/g);
                if (badchars) {
                    throw new Error(`Found ${badchars.length} unallowed characters in time entry: "${badchars.toString()}"`);
                }
                var nounit = human.match(/\d+$/g);
                if (nounit) {
                    throw new Error(`Numeric value ${nounit[0]} lacks time unit`);
                }
                var nonum = human.match(/^[a-z]+/g);
                if (nonum) {
                    throw new Error(`${nonum[0]} is not a valid time duration`);
                }
                
                var humanTokens = human.match(/\d+[a-z]+/g);
                var msecs = humanTokens
                    .reduce(function(prev, tok){
                        var tokPart = tok.match(/(\d+)([a-z]+)/);
                        if (! (tokPart[2] in TIMEMULTIPLIER)) {
                            // throw new Error("Unable to convert input " + humanTime + ": invalid time unit " + tokPart[0]);
                            throw new Error(`Unable to convert input ${humanTime}: invalid time unit "${tokPart[2]}"`);
                        }  
                        return parseFloat(tokPart[1]) * TIMEMULTIPLIER[tokPart[2]] + prev;
                    }, 0);
                return msecs;
            },

            validateDuration: function(durationStr){
                try {
                    utils.parseDuration(durationStr);
                } catch (e) {
                    return e.message;
                }
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
                if (isNaN(+port)) {
                    return $translate.instant("port_number_invalid");
                }
                if(+port < 1 || +port > 65535){
                    return $translate.instant("port_number_invalid_range");
                }

                return null;
            },

            validateRAMThresholdLimit: function(limitStr){
                if(isNaN(limitStr) || limitStr === undefined){
                    return "Invalid RAM threshold Limit value";
                }
                if(limitStr < 0){
                    return "RAM threshold Limit cannot be less than 0%";
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
            },

            NgBitset: NgBitset,

            arrayEmpty: function(array) {
                return typeof array === "undefined" || array === null || array.length <= 0;
            },

            escapeHTML: function (snippet) {
                return snippet
                            .replace(/&/g, "&amp;")
                            .replace(/</g, "&lt;")
                            .replace(/>/g, "&gt;")
                            .replace(/"/g, "&quot;")
                            .replace(/'/g, "&#039;");
            }
       };

        return utils;
    }]);
})();
