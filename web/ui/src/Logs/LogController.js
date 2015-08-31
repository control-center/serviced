/* LogController
 * displays kibaba log iframe
 */
(function() {
    'use strict';

    controlplane.controller("LogController", ["$scope", "authService",
    function($scope, authService) {
        authService.checkLogin($scope);
        $scope.breadcrumbs = [
            { label: 'breadcrumb_logs', itemClass: 'active' }
        ];

        $scope.$emit("ready");

        // force log iframe to fill screen
        setInterval(function() {
            var logsframe = document.getElementById("logsframe");

            if (logsframe && logsframe.contentWindow.document.body){
                var h = logsframe.contentWindow.document.body.clientHeight;
                logsframe.height = h + "px";
            }

        }, 100);
    }]);
})();
