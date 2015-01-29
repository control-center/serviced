/* global: $ */
/* jshint multistr: true */
(function() {
    'use strict';

    angular.module('ui.datetimepicker', []).directive('datetimepicker', [
    function () {
        return {
            restrict: 'A',
            require: 'ngModel',
            link: function (scope, element, attrs, ngModelCtrl) {
                // wait a tick before init because calling directive
                // may not have completed its init yet
                setTimeout(function(){
                    var options = {};
                    if (attrs.dateoptions && scope[attrs.dateoptions]) {
                        options = scope[attrs.dateoptions];
                    }
                    element.datetimepicker(options);
                    element.bind('blur keyup change', function(){
                        var model = attrs.ngModel;
                        if (model.indexOf(".") > -1){
                            scope[model.replace(/\.[^.]*/, "")][model.replace(/[^.]*\./, "")] = element.val();
                        } else {
                            scope[model] = element.val();
                        }
                    });
                }, 0);
            }
        };
    }]);


})();
