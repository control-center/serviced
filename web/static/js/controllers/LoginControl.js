function LoginControl($scope, $location, $notification, $translate, authService) {

    if(navigator.userAgent.indexOf("Trident") > -1 && navigator.userAgent.indexOf("MSIE 7.0") > -1){
        $notification.create("", $translate.instant("compatibility_mode"), $("#loginNotifications")).warning(false);
    }

    $scope.brand_label = "CONTROL CENTER";

    enableLoginButton();

    // Makes XHR POST with contents of login form
    $scope.login = function() {
        disableLoginButton();

        var creds = { "Username": $scope.username, "Password": $scope.password };
        authService.login(creds, function(){
            enableLoginButton();
            // Redirect to main page
            $location.path('/apps');
        }, function(){
            enableLoginButton();
            // display fail message to user
            $notification.create("", $translate.instant("login_fail"), $("#loginNotifications")).error();
        });
    };

    function enableLoginButton(){
        $scope.login_button_text = $translate.instant("log_in");
        $scope.loginDisabled = false;
    }

    function disableLoginButton(){
        $scope.login_button_text = $translate.instant("logging_in"); 
        $scope.loginDisabled = true;
    }
}
