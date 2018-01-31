/* LoginController
 * login page
 */
(function() {
    'use strict';

    controlplane.controller("LoginController", ["$scope", "$location", "$notification", "$translate", "authService", "resourcesFactory",
    function($scope, $location, $notification, $translate, authService, resourcesFactory) {

        if(navigator.userAgent.indexOf("Trident") > -1 && navigator.userAgent.indexOf("MSIE 7.0") > -1){
            $notification.create("", $translate.instant("compatibility_mode"), $("#loginNotifications")).warning(false);
        }

        enableLoginButton();

        $scope.$emit("ready");

        $scope.version = "";
        resourcesFactory.getVersion().success(function(data){
            $scope.version = data.Version;
        });

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
            $scope.loginButtonText = "log_in";
            $scope.loginDisabled = false;
        }

        function disableLoginButton(){
            $scope.loginButtonText = "logging_in";
            $scope.loginDisabled = true;
        }
    }]);
})();
