/* serviceActionssDirective
 * directive for displaying processing state of a service/instance
 */
(function() {
    'use strict';

    angular.module('serviceActions', [])
    .directive("serviceActions", ["svcActions",
    function(svcActions){

        class ServiceActionsController {

            constructor() {
                // howdy
            }

            actionClick(action, isEnabled) {
                let service = this.service;
            }

            start(force) {
                this.service.start();

            }

            restart(force) {
                this.service.restart();
            }

            stop(force) {
                this.service.stop();
            }

            cancel() {
                
            }

            update(){

                if (this.service.isContainer()) {
                    this.showDefaultActions();
                } else {
                    this.showValidActions(this.service.currentState);
                }

            }


        }

        let actsTemplate = `
        <div class="svc-actionbar">

            <div class="svcactions-starts">
                <button 
                    data-valid="stopping,stopped"
                    ng-click="vm.start(false)" 
                    class="btn btn-link action svcactions svcactions-start"
                >
                    <i class="glyphicon glyphicon-play"></i>
                    <span translate>start</span>
                </button>
                <button 
                    data-valid="pending_start"
                    ng-click="vm.start(true)" 
                    class="btn btn-link action svcactions vcactions-startnow"
                >
                    <i class="glyphicon glyphicon-play"></i>
                    <span translate>btn_start_now</span>
                </button>
                <button 
                    data-valid="running"
                    ng-click="vm.restart(false)" 
                    class="btn btn-link action svcactions svcactions-restart"
                >
                    <i class="glyphicon glyphicon-refresh"></i>
                    <span translate>action_restart</span>
                </button>
                <button 
                    data-valid="pending_restart"
                    ng-click="vm.restart(true)" 
                    class="btn btn-link action svcactions svcactions-restartnow"
                >
                    <i class="glyphicon glyphicon-refresh"></i>
                    <span translate>btn_restart_now</span>
                </button>
                <span class="svcactions svcactions-na">
                    <i class="glyphicon glyphicon-play"></i>
                    <span translate>start</span>
                </span>
            </div>

            <div class="svcactions-stops">
                <button 
                    data-valid="starting,running,pending_restart,restarting"
                    ng-click="vm.stop(false)" 
                    class="btn btn-link action svcactions svcactions-stop"
                >
                    <i class="glyphicon glyphicon-stop"></i>
                    <span translate>stop</span>
                </button>
                <button 
                    data-valid="pending_stop"
                    ng-click="vm.stop(true)" 
                    class="btn btn-link action svcactions svcactions-stopnow"
                >
                    <i class="glyphicon glyphicon-stop"></i>
                    <span translate>btn_stop_now</span>
                </button>
                <span class="svcactions svcactions-na">
                    <i class="glyphicon glyphicon-stop"></i>
                    <span translate>stop</span>
                </span>
            </div>

            <div class="svcactions-cancels">
                <button 
                    data-valid="pending_stop,pending_start,pending_restart"
                    ng-click="vm.cancel()" 
                    class="btn btn-link action svcactions svcactions-cancel"
                >
                    <i class="glyphicon glyphicon-remove"></i>
                    <span translate>btn_cancel</span>
                </button>
                <span class="svcactions svcactions-na">
                    <i class="glyphicon glyphicon-remove"></i>
                    <span translate>btn_cancel</span>
                </span>
            </div>

        </div>
        `;

        return {
            restrict: "E",
            template: actsTemplate,
            scope: {
                service: "="
            },
            controller: ServiceActionsController,
            controllerAs: "vm",
            bindToController: true,            
            link: function($scope, element, attrs){

                let allButtonEls  = element.find(".svcactions");
                let startActions  = element.find(".svcactions-starts");
                let stopActions   = element.find(".svcactions-stops");
                let cancelActions = element.find(".svcactions-cancels");

                let showValidActions = function(state) {
                    allButtonEls.hide();

                    let el = startActions.find("[data-valid*=" + state + "]");
                    if (el.length) {
                        el.show();
                    } else {
                        startActions.find(".svcactions-na").show();
                    }

                    el = stopActions.find("[data-valid*=" + state + "]");
                    if (el.length) {
                        el.show();
                    } else {
                        stopActions.find(".svcactions-na").show();
                    }

                    el = cancelActions.find("[data-valid*=" + state + "]");
                    if (el.length) {
                        el.show();
                    } else {
                        cancelActions.find(".svcactions-na").show();
                    }

                };


                let showDefaultActions = function() {
                    allButtonEls.hide();
                    startActions.find(".svcactions-start").show();                    
                    stopActions.find(".svcactions-stop").show();                    
                };


                $scope.vm.showValidActions = showValidActions;
                $scope.vm.showDefaultActions = showDefaultActions;

                // if status object updates, update icon
                $scope.$watch("vm.service.currentState", $scope.vm.update.bind($scope.vm));

                // TODO - cleanup watch
                $scope.$on("$destroy", function(){});

        }};
    }]);
})();
