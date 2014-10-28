function LoginControl($scope, $location, $notification, $translate, authService) {

    if(navigator.userAgent.indexOf("Trident") > -1 && navigator.userAgent.indexOf("MSIE 7.0") > -1){
        $notification.create("", $translate.instant("compatibility_mode"), $("#loginNotifications")).warning(false);
    }

    $scope.brand_label = "CONTROL CENTER";
    $scope.login_button_text = "Log In";

    // Makes XHR POST with contents of login form
    $scope.login = function() {
        var creds = { "Username": $scope.username, "Password": $scope.password };
        authService.login(creds, function(){
            // Redirect to main page
            $location.path('/apps');
        }, function(){
            // display fail message to user
            $notification.create("", $translate.instant("login_fail"), $("#loginNotifications")).error();
        });
    };
}
