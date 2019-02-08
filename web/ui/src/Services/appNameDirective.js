/* appNameDirective
 * directive for displaying name and status of a top-level service (application)
 */
(function() {
    'use strict';

    angular.module('appName', [])
    .directive('appName', ["resourcesFactory", function(resourcesFactory) {
        var template = `
            <div class="appName">
                <div ng-show="app.isEmergencyShutdown()" ng-class="app.getStatusClass()">
                    <i class="healthIcon glyphicon"></i>
                </div>
                <span ng-if="!app.service.deploying" ng-click="app.routeToService()" class="link" ng-class="app.getStatusClass()">{{app.service.name}}<span class="version" ng-show="app.service.model.Version"> (v{{app.service.model.Version}})</span></span>
                <span ng-if="app.service.deploying">{{app.service.name}}<span class="version" ng-show="app.service.model.Version"> (v{{app.service.model.Version}})</span></span>
            </div>
            `;


        let popoverContentTemplate = m => {
            return `
            <div class="emergencyTooltip">
              <h3>Emergency Shutdown</h3>
              <div>due to low disk space.</div>
              <a href="https://help.zenoss.com/zsd/cc/administering-control-center/emergency-shutdown-of-services/resetting-emergency-shutdown-flags" type="button" class="btn btn-danger">
                <span ng-show="icon" class="glyphicon glyphicon-book"></span>
                More Info
              </a>
            </div>
            `;
        };

        class Controller {
            constructor(){
                // hi.
            }

            isEmergencyShutdown(){
              if (!this.service) {
                return false;
              }
              return this.service.emergencyShutdown;
            }

            routeToService() {
                if (this.service.isIsvc()) {
                    resourcesFactory.routeToInternalServices();
                } else {
                    resourcesFactory.routeToService(this.service.id);
                }
            }

            getStatusClass(){
                if (!this.service.status) {
                  return "";
                }
                return this.service.status.status;
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
                service: "=",
            },
            controller: Controller,
            controllerAs: "app",
            bindToController: true,
            template: template,
            link: function(scope, element, attrs) {
                let $appname = $(element).find(".appName");
                $appname.popover({
                    trigger: "hover",
                    placement: "bottom",
                    delay: {
                        show: 100,
                        hide: 400
                    },
                    html: true
                });

                // NOTE: directly accessing the bootstrap popover
                // data object here.
                scope.app.popover = $appname.data("bs.popover");

                // Don't show a popover for services that aren't emergency
                // shutdown.
                $appname.on('show.bs.popover', function() {
                    return scope.app.isEmergencyShutdown() || false;
                });

                // Override the popover's default hide behavior so that it stays
                // visible when the mouse is in the popover itself, allowing
                // clickable links.
                $appname.on('hide.bs.popover', function() {
                    let popover = scope.app.popover;
                    let container = popover.$tip;
                    if (container.is(':visible') && container.is(':hover')) {
                        container.one('mouseleave', function() {
                            setTimeout(function() {
                                popover.hide();
                            }, popover.options.delay.hide);
                        });
                        return false;
                    }
                });

                scope.app.updatePopover();

                scope.$watch(function(){
                    return scope.app.isEmergencyShutdown();
                }, function(newVal, oldVal) {
                    scope.app.updatePopover();
                });
            }
        };
  }]);

}());

