/* log.js
 * logging, pure of heart.
 */
(function(){
    "use strict";

    let getCallingFn = function(index){
        try {
            let e = new Error("stack");
            return e.stack
                .split("\n")[index]
                .trim()
                .replace("at ", "");
        } catch(e) {
            return "";
        }
    };

    const DEBUG = "debug",
          LOG = "log",
          INFO = "info",
          WARN = "warn",
          ERROR = "error";

    const logLevels = [DEBUG, LOG, INFO, WARN, ERROR];

    // number of steps up the stack to get out of the
    // logging library and back to the actual caller
    const DEBUG_DEPTH = 3;

    let $log;

    const logFunctions = {
        [DEBUG]: function(...args){
            $log.debug("[DEBUG: "+ getCallingFn(DEBUG_DEPTH) +"]\n", ...args);
        },

        [LOG]: function(...args){
            $log.log("[LOG: "+ getCallingFn(DEBUG_DEPTH) +"]\n", ...args);
        },

        [INFO]: function(...args){
            $log.info("[INFO: "+ getCallingFn(DEBUG_DEPTH) +"]\n", ...args);
        },

        [WARN]: function(...args){
            $log.warn("[WARN: "+ getCallingFn(DEBUG_DEPTH) +"]\n", ...args);
        },

        [ERROR]: function(...args){
            $log.error("[ERROR: "+ getCallingFn(DEBUG_DEPTH) +"]\n", ...args);
        }
    };

    const noop = ()=>{};

    class Log {
        constructor(){
            this.level = {
                [DEBUG]: DEBUG,
                [LOG]: LOG,
                [INFO]: INFO,
                [WARN]: WARN,
                [ERROR]: ERROR
            };
            this.setLogLevel(WARN);
        }

        setLogLevel(newLevel){
            let newLevelIndex = logLevels.indexOf(newLevel);

            if(newLevelIndex === -1){
                $log.warn(`could not set log level '${newLevel}'`);
                return;
            }

            logLevels.forEach((level, i) => {
                if(i >= newLevelIndex){
                    this[level] = logFunctions[level];
                } else {
                    this[level] = noop;
                }
            });
        }

    }

    angular.module("log", [])
    .factory("log", ["$log", function(_$log){
        $log = _$log;
        return new Log();
    }]);
})();
