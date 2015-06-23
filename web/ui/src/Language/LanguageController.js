/* LanguageController
 * toggle selected language
 */
(function() {
    'use strict';

    controlplane.controller("LanguageController", ["$scope", "$cookies", "$translate", "miscUtils",
    function($scope, $cookies, $translate, utils) {
        $scope.name = 'language';
        $scope.setUserLanguage = function() {
            console.log('User clicked', $scope.user.language);
            $cookies.Language = $scope.user.language;
            utils.updateLanguage($scope, $cookies, $translate);
        };
        $scope.getLanguageClass = function(language) {
            return ($scope.user.language === language)? 'btn btn-primary active' : 'btn btn-primary';
        };
    }]);
})();
