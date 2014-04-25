function DevControl($scope, $cookieStore, authService) {
    authService.checkLogin($scope);
    $scope.name = "developercontrol";

    var updateDevMode = function() {
        if ($scope.devmode.enabled) {
            $scope.devmode.enabledClass = 'btn btn-success active';
            $scope.devmode.enabledText = 'Enabled';
            $scope.devmode.disabledClass = 'btn btn-default off';
            $scope.devmode.disabledText = '\xA0'; // &nbsp;
        } else {
            $scope.devmode.enabledClass = 'btn btn-default off';
            $scope.devmode.enabledText = '\xA0';
            $scope.devmode.disabledClass = 'btn btn-danger active';
            $scope.devmode.disabledText = 'Disabled'; // &nbsp;
        }
    };
    $scope.devmode = {
        enabled: $cookieStore.get('ZDevMode')
    };
    $scope.setDevMode = function(enabled) {
        $scope.devmode.enabled = enabled;
        $cookieStore.put('ZDevMode', enabled);
        updateDevMode();
    };
    updateDevMode();
}
