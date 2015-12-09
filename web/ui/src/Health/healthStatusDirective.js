
/* healthStatusDirective
 * directive for displaying health of a service/instance
 * using popover details
 */
(function() {
    'use strict';

    angular.module('healthStatus', [])
    .directive("healthStatus", [ "$translate",
    function($translate) {
        var linker = function($scope, element, attrs) {
            // Because we don't need to track status, we just need to enable the
            // bootstrap popover.
            // Set the popup with the content data.
            $(element).popover({
                trigger: "hover",
                placement: "top",
                delay: 0,
                content: $translate.instant("vhost_unavailable"),
            });
        };
        return {
            restrict: "E",
            link: linker
        };
    }]);
})();
