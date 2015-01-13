function LanguageControl($scope, $cookies, $translate) {
    $scope.name = 'language';
    $scope.setUserLanguage = function() {
        console.log('User clicked %s', $scope.user.language);
        $cookies.Language = $scope.user.language;
        updateLanguage($scope, $cookies, $translate);
    };
    $scope.getLanguageClass = function(language) {
        return ($scope.user.language === language)? 'btn btn-primary active' : 'btn btn-primary';
    };
}
