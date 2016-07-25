/* StatusController
 * Displays system stats
 */
(function() {
    'use strict';

    controlplane.controller("StatusController", ["$scope", "authService",
    function($scope, authService) {
        authService.checkLogin($scope);
        $scope.breadcrumbs = [
            { label: 'breadcrumb_status', itemClass: 'active' }
        ];

        $scope.$emit("ready");

    }]);
})();
