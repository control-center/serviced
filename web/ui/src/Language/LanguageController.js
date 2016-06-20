/* LanguageController
 * toggle selected language
 */
(function() {
    'use strict';

    controlplane.controller("LanguageController", ["$scope", "servicedConfig", "$translate", "miscUtils", "log",
    function($scope, servicedConfig, $translate, utils, log) {
        $scope.name = 'language';
        $scope.setUserLanguage = function() {
            log.log('User clicked', $scope.user.language);
            servicedConfig.set("Language",$scope.user.language);
            utils.updateLanguage($scope, servicedConfig, $translate);
        };
        $scope.getLanguageClass = function(language) {
            return ($scope.user.language === language)? 'btn btn-primary active' : 'btn btn-primary';
        };
    }]);
})();
