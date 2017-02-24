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
                //let service = this.service;
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
                this.service.cancelPending();
            }

            update(){
                if (this.service && !this.service.isContainer()) {
                    this.showValidActions(this.service.currentState);
                } else {
                    this.showDefaultActions();
                }
            }

        }

        let actsTemplate = `
        <div class="svc-actionbar">

            <div class="svcactions-starts">
                <button
                    class="btn btn-link action svcactions svcactions-start"
                    ng-click="vm.start(false)"
                    data-valid="stopping,stopped,unknown"
                >
                    <i class="glyphicon glyphicon-play"></i>
                    <span translate>start</span>
                </button>
                <button
                    class="btn btn-link action svcactions svcactions-startnow"
                    ng-click="vm.start(true)"
                    data-valid="pending_start"
                >
                    <i class="glyphicon glyphicon-play"></i>
                    <span translate>btn_start_now</span>
                </button>
                <button
                    class="btn btn-link action svcactions svcactions-restart"
                    ng-click="vm.restart(false)"
                    data-valid="started"
                >
                    <i class="glyphicon glyphicon-refresh"></i>
                    <span translate>action_restart</span>
                </button>
                <button
                    class="btn btn-link action svcactions svcactions-restartnow"
                    ng-click="vm.restart(true)"
                    data-valid="pending_restart"
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
                    class="btn btn-link action svcactions svcactions-stop"
                    ng-click="vm.stop(false)"
                    data-valid="starting,started,pending_restart,restarting,unknown"
                >
                    <i class="glyphicon glyphicon-stop"></i>
                    <span translate>stop</span>
                </button>
                <button
                    class="btn btn-link action svcactions svcactions-stopnow"
                    ng-click="vm.stop(true)"
                    data-valid="pending_stop"
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
                    class="btn btn-link action svcactions svcactions-cancel"
                    ng-click="vm.cancel()"
                    data-valid="pending_stop,pending_start,pending_restart"
                >
                    <i class="glyphicon glyphicon-remove"></i>
                    <span translate>btn_cancel</span>
                </button>
                <span class="svcactions svcactions-na">
                    <i class="glyphicon glyphicon-remove"></i>
                    <span translate>btn_cancel</span>
                </span>
                <button
                    class="btn btn-link action svcactions svcactions-container-restart"
                    ng-click="vm.restart(false)"
                    data-valid=""
                >
                    <i class="glyphicon glyphicon-refresh"></i>
                    <span translate>action_restart</span>
                </button>
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
                    cancelActions.find(".svcactions-container-restart").show();
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
