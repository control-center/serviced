/* authService.js
 * determine if user is authorized
 */ 
(function(){
    "use strict";

    angular.module("authService", [])
    .factory("authService", ["$cookies", "$cookieStore", "$location", "$http", "$notification", "miscUtils",
    function($cookies, $cookieStore, $location, $http, $notification, utils) {
        var loggedIn = false;
        var userName = null;

        var setLoggedIn = function(truth, username) {
            loggedIn = truth;
            userName = username;
        };
        return {

            /*
             * Called when we have positively determined that a user is or is not
             * logged in.
             *
             * @param {boolean} truth Whether the user is logged in.
             */
            setLoggedIn: setLoggedIn,

            login: function(creds, successCallback, failCallback){
                $http.post('/login', creds).
                    success(function(data, status) {
                        // Ensure that the auth service knows that we are logged in
                        setLoggedIn(true, creds.Username);

                        successCallback();
                    }).
                    error(function(data, status) {
                        // Ensure that the auth service knows that the login failed
                        setLoggedIn(false);

                        failCallback();
                    });
            },

            logout: function(){
                $http.delete('/login').
                    success(function(data, status) {
                        // On successful logout, redirect to /login
                        $location.path('/login');
                    }).
                    error(function(data, status) {
                        // On failure to logout, note the error
                        // TODO error screen
                        console.error('Unable to log out. Were you logged in to begin with?');
                    });
            },

            /*
             * Check whether the user appears to be logged in. Update path if not.
             *
             * @param {object} scope The 'loggedIn' property will be set if true
             */
            checkLogin: function($scope) {
                $scope.dev = $cookieStore.get('ZDevMode');
                if (loggedIn) {
                    $scope.loggedIn = true;
                    $scope.user = {
                        username: $cookies.ZUsername
                    };
                    return;
                }
                if ($cookies.ZCPToken) {
                    loggedIn = true;
                    $scope.loggedIn = true;
                    $scope.user = {
                        username: $cookies.ZUsername
                    };
                } else {
                    utils.unauthorized($location);
                }
            }
        };
    }]);
})();
