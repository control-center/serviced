/* global angular, console, $ */
/* jshint multistr: true */
(function() {
    'use strict';
angular.module('ui.timepicker', [])

    .value('uiTimepickerConfig', {
        'step' : 15
    })

    .directive('uiTimepicker', ['uiTimepickerConfig','$parse', function(uiTimepickerConfig, $parse) {
        return {
            restrict: 'A',
            require: 'ngModel',
            priority: 1,
            link: function(scope, element, attrs, ngModel) {
                'use strict';
                var config = angular.copy(uiTimepickerConfig);

                ngModel.$render = function () {
                    var date = ngModel.$modelValue;
                    if ( angular.isDefined(date) && date !== null && !angular.isDate(date) ) {
                        throw new Error('ng-Model value must be a Date object - currently it is a ' + typeof date + '.');
                    }
                    if (!element.is(':focus')) {
                        element.timepicker('setTime', date);
                    }
                };

                scope.$watch(attrs.ngModel, function() {
                    ngModel.$render();
                }, true);

                config.appendTo = element.parent();

                element.timepicker(
                    angular.extend(
                        config, attrs.uiTimepicker ?
                            $parse(attrs.uiTimepicker)(scope):
                            {}
                    )
                );

                if(element.is('input'))  {
                    ngModel.$parsers.unshift(function(){
                        var date = element.timepicker('getTime', ngModel.$modelValue);
                        return date;
                    });
                } else {
                    element.on('changeTime', function() {
                        scope.$evalAsync(function() {
                            var date = element.timepicker('getTime', ngModel.$modelValue);
                            ngModel.$setViewValue(date);
                        });
                    });
                }
            }
        };
    }]);
})();
