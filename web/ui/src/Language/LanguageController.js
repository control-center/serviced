/* LanguageController
 * toggle selected language
 */
(function() {
    'use strict';

    controlplane.controller("LanguageController", ["$scope", "$cookies", "$translate", "miscUtils", "log",
    function($scope, $cookies, $translate, utils, log) {
        $scope.name = 'language';
        $scope.setUserLanguage = function() {
            log.log('User clicked', $scope.user.language);
            $cookies.put("Language",$scope.user.language);
            utils.updateLanguage($scope, $cookies, $translate);
        };
        $scope.getLanguageClass = function(language) {
            return ($scope.user.language === language)? 'btn btn-primary active' : 'btn btn-primary';
        };
    }]);
})();
