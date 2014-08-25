function LoginControl($scope, $http, $location, $notification, $translate, authService) {
    $scope.brand_label = "CONTROL CENTER";
    $scope.login_button_text = "Log In";

    // Makes XHR POST with contents of login form
    $scope.login = function() {
        var creds = { "Username": $scope.username, "Password": $scope.password };
        $http.post('/login', creds).
            success(function(data, status) {
                // Ensure that the auth service knows that we are logged in
                authService.login(true, $scope.username);
                // Redirect to main page
                $location.path('/apps');
            }).
            error(function(data, status) {
                $notification.create("", $translate.instant("login_fail"), $("#loginNotifications")).error();
                // Ensure that the auth service knows that the login failed
                authService.login(false);
            });
    };
}