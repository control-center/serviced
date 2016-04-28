/* areUIReady.js
 * provides a means of locking the UI to
 * prevent interaction when it is not ready
 */
(function(){
    "use strict";

    let rgx = /\(([^)]+)\)/;
    // gets name of the calling function from
    // stack trace. this is hardcoded to step up
    // 3 levels
    let getCallingFn = function(){
        try {
            let e = new Error("stack");
            let match = e.stack.split("\n")[3].match(rgx);
            return match && match[1];
        } catch(e) {
            return;
        }
    };

    class ReadyState {
        constructor(log){
            this.log = log;
            this.locked = false;
            this.lockCount = 0;
            this.onLock = ()=>{};
            this.onUnlock = ()=>{};
        }

        lock(){
            if(!this.locked){
                this.locked = true;
                this.onLock(this);
            }
            this.lockCount++;
            this.log.info("lock count is now", this.lockCount, getCallingFn());
        }

        unlock(){
            /*
            // NOTE - uncomment to reimplement lock counting
            if(this.lockCount === 0){
                // can't unlock because none are locked
                this.log.info("unable to unlock UI, UI is not locked");
                return;
            }*/

            this.lockCount--;
            this.log.info("lock count is now", this.lockCount);

            // NOTE - uncomment to reimplement lock counting
            //if(this.lockCount === 0){
                this.log.info("ui is unlocked!", getCallingFn());
                this.locked = false;
                this.onUnlock(this);
            //}
        }

        isLocked(){
            return this.locked;
        }
    }

    angular.module("areUIReady", [])
        .service("areUIReady", ["log", ReadyState]);
})();
