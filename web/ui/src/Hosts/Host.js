(function () {
    'use strict';

    // share angular services outside of angular context
    let utils;

    controlplane.factory('Host', HostFactory);

    class Host {

        constructor(host) {
            this.name = host.Name;
            this.id = host.ID;
            this.model = Object.freeze(host);

            this._status = {
                Active: false,
                Authenticated: false,
                MemoryUsage: {Cur: 0, Max: 0, Avg: 0}
            };

            this.touch();
        }

        touch() {
            this.lastUpdate = new Date().getTime();
        }

        resourcesGood() {
            return this.RAMAverage <= this.RAMLimitBytes;
        }

        get status() {
            return this._status;
        }

        set status(value) {
            this._status = value;
            this.touch();
        }

        RAMIsPercent() {
            return this.RAMLimit.endsWith("%");
        }

        get RAMLimit() {
            if(!this.model){
                return undefined;
            }

            return this.model.RAMLimit || "100%";
        }

        get RAMLimitBytes(){
            if(this.RAMIsPercent()){
                return + this.RAMLimit.slice(0,-1) * this.model.Memory * 0.01;
            } else {
                return utils.parseEngineeringNotation(this.RAMLimit);
            }
        }

        get RAMLast() {
            return this._status.MemoryUsage.Cur || 0;
        }

        get RAMMax() {
            return this._status.MemoryUsage.Max || 0;
        }

        get RAMAverage() {
            return this._status.MemoryUsage.Avg || 0;
        }
    }

    HostFactory.$inject = ['miscUtils'];
    function HostFactory(_utils) {
        utils = _utils;
        return Host;
    }

})();

