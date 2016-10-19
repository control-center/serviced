/* hostIconDirective
 * directive for displaying status of a host
 */
(function() {
    'use strict';

    angular.module('hostIcon', [])
    .directive('hostIcon', [function() {
        var template = `
            <div ng-class="vm.getHostStatusClass()" style="position: relative; height: 22px;">
                <i class="healthIcon glyphicon"></i>
            </div>`;

        let popoverContentTemplate = m => {
            return `
                <div class='healthTooltipDetailRow ${m.getHostActiveStatusClass()}'>
                    <i class='healthIcon glyphicon'></i>
                    <div class='healthTooltipDetailName'>Active</div>
                </div>
                <div class='healthTooltipDetailRow ${m.getHostAuthStatusClass()}'>
                    <i class='healthIcon glyphicon'></i>
                    <div class='healthTooltipDetailName'>Authenticated</div>
                </div>
            `;
        };

        class Controller {
            constructor(){
                // hi.
            }
            _getHostStatus(){
                if(!this.host){
                    return {active: null, authed: null};
                }

                let status = this.getHostStatus(this.host.id);

                if(!status){
                    return {active: null, authed: null};
                }

                let active = status.Active,
                    authed = status.Authenticated;

                return {active, authed};
            }

            getHostStatusClass(){
                let {active, authed} = this._getHostStatus();

                // stuff hasnt loaded, so unknown
                if(active === null && authed === null){
                    return "unknown";
                }

                // connected and authenticated
                if(active && authed){
                    return "passed";

                // connected but not yet authenticated
                } else if(active && !authed){
                    // TODO - something more clearly related to auth
                    return "failed";

                // not connected
                } else {
                    return "not_running";
                }
            }

            getHostActiveStatusClass(){
                let {active} = this._getHostStatus(),
                    status;

                if(active === true){
                    status = "passed";
                } else if(active === false){
                    status = "not_running";
                } else {
                    status = "unknown";
                }

                return status;
            }

            getHostAuthStatusClass(){
                let {active, authed} = this._getHostStatus(),
                    status;

                if(authed === true){
                    status = "passed";
                } else if(active === true && authed === false){
                    status = "failed";
                } else if(active === false && authed === false){
                    status = "not_running";
                } else {
                    status = "unknown";
                }

                return status;
            }

            updatePopover(){
                let p = this.popover;
                p.options.content = popoverContentTemplate(this);
                p.setContent();

                // if the popover is currently visible, update
                // it immediately, but turn off animation to
                // prevent it fading in
                if(p.$tip.is(":visible")){
                    p.options.animation = false;
                    p.show();
                    p.options.animation = true;
                }
            }
        }

        return {
            restrict: "EA",
            scope: {
                host: "=",
                getHostStatus: "="
            },
            controller: Controller,
            controllerAs: "vm",
            bindToController: true,
            template: template,
            link: function(scope, element, attrs) {
                let $icon = $(element).find(".healthIcon");
                $icon.popover({
                    trigger: "hover",
                    placement: "right",
                    delay: 0,
                    html: true
                });
                // NOTE: directly accessing the bootstrap popover
                // data object here.
                scope.vm.popover = $icon.data("bs.popover");
                scope.vm.updatePopover();

                // use a bitfield to describe host state as a
                // single value to determine if we need to
                // update the view
                var ACTIVE = 1 << 1,
                    AUTHED = 1 << 2;

                scope.$watch(function(){
                    let {active, authed} = scope.vm._getHostStatus(),
                        val = 0;

                    // no results have come back yet
                    if(active === null && authed === null){
                        return -1;
                    }

                    // results, so lets smoosh em
                    // into a single value
                    if(active){
                        val = val ^ ACTIVE;
                    }
                    if(authed){
                        val = val ^ AUTHED;
                    }

                    return val;
                }, function(newVal, oldVal){
                    scope.vm.updatePopover();
                });
            }
        };
  }]);

}());
