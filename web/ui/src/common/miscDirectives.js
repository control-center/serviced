/* miscDirectives.js
 * a place for miscellaneous directives
 */
(function(){
    "use strict";

    controlplane
    /**
     * This is a fix for https://jira.zenoss.com/browse/ZEN-10263
     * It makes sure that inputs that are filled in by autofill (like when the browser remembers the password)
     * are updated in the $scope. See the partials/login.html for an example
     **/
    .directive('formAutofillFix', [
    function() {
        return function(scope, elem, attrs) {
            // Fixes Chrome bug: https://groups.google.com/forum/#!topic/angular/6NlucSskQjY
            elem.prop('method', 'POST');

            // Fix autofill issues where Angular doesn't know about autofilled inputs
            if(attrs.ngSubmit) {
                window.setTimeout(function() {
                    elem.unbind('submit').submit(function(e) {
                        e.preventDefault();
                        elem.find('input, textarea, select').trigger('input').trigger('change').trigger('keydown');
                        scope.$apply(attrs.ngSubmit);
                    });
                }, 0);
            }
        };
    }])
    .directive('popover', [
    function(){
        return function(scope, elem, attrs){
            $(elem).popover({
                title: attrs.popoverTitle,
                trigger: "hover",
                html: true,
                content: attrs.popover
            });
        };
    }])
    .directive('scroll', ["$rootScope", "$window", "$timeout",
    function($rootScope, $window, $timeout) {
        return {
            link: function(scope, elem, attr) {
                $window = angular.element($window);
                var handler = function() {
                    var winEdge, elEdge, dataHidden;
                    winEdge = $window.height() + $window.scrollTop();
                    elEdge = elem.offset().top + elem.height();
                    dataHidden = elEdge - winEdge;
                    if (dataHidden < parseInt(attr.scrollSize, 10)) {
                        if ($rootScope.$$phase) {
                            if (scope.$eval(attr.scroll)) {
                                $timeout(handler, 100);
                            }
                        } else {
                            if (scope.$apply(attr.scroll)) {
                                $timeout(handler, 100);
                            }
                        }
                    }
                };
                if (attr.scrollHandlerObj && attr.scrollHandlerField) {
                    var obj = scope[attr.scrollHandlerObj];
                    obj[attr.scrollHandlerField] = handler;
                }
                $window.on('scroll', handler);
                $window.on('resize', handler);
                scope.$on('$destroy', function() {
                    $window.off('scroll', handler);
                    $window.off('resize', handler);
                    return true;
                });
                return $timeout(function() {
                    return handler();
                }, 100);
            }
        };
    }]);
})();
