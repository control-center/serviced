/* authService.js
 * determine if user is authorized
 */
(function(){
    "use strict";

    angular.module("authService", ["auth0.auth0"])
    .factory("authService", ["angularAuth0", "$cookies", "$cookieStore", "$location", "$http", "$notification", "miscUtils", "log",
    function(angularAuth0, $cookies, $cookieStore, $location, $http, $notification, utils, log) {
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

            auth0login: function () {
                angularAuth0.authorize();
            },

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
                        window.sessionStorage.removeItem("auth0AccessToken");
                        window.sessionStorage.removeItem("auth0IDToken");
                        let redirectloc = '/';
                        if (utils.useAuth0()) {
                            let returnloc = encodeURIComponent(window.location.origin + '/');
                            redirectloc = 'https://' + window.Auth0Config.Auth0Domain + '/v2/logout' +
                                '?returnTo=' + returnloc +
                                '&client_id=' + window.Auth0Config.Auth0ClientID;
                        }
                        // On successful logout, redirect to /
                        window.location = redirectloc;
                    }).
                    error(function(data, status) {
                        // On failure to logout, note the error
                        // TODO error screen
                        log.error('Unable to log out. Were you logged in to begin with?');
                    });
            },

            /*
             * Check whether the user appears to be logged in. Update path if not.
             *
             * @param {object} scope The 'loggedIn' property will be set if true
             */
            checkLogin: function($scope) {
                if (utils.useAuth0()) {
                    var at = window.sessionStorage.getItem("auth0AccessToken");
                    var it = window.sessionStorage.getItem("auth0IDToken");
                    if (at && it) {
                        $scope.loggedIn = true;
                        $scope.user = {
                            username: "successful auth0 login"
                        };
                        return;
                    }
                } else {
                    $scope.dev = $cookieStore.get("ZDevMode");
                    if (loggedIn || $cookies.get("ZCPToken")) {
                        $scope.loggedIn = true;
                        $scope.user = {
                            username: $cookies.get("ZUsername")
                        };
                        return;
                    }
                }
                utils.unauthorized($scope, $location);
            }
        };
    }]).config(config);

    config.$inject = [
        'angularAuth0Provider'
    ];

    function config(angularAuth0Provider) {
        // Initialization for the angular-auth0 library
        angularAuth0Provider.init({
            domain: window.Auth0Config.Auth0Domain,
            clientID: window.Auth0Config.Auth0ClientID,
            redirectUri: window.location.origin + "/static/auth0callback.html",
            audience: window.Auth0Config.Auth0Audience,
            responseType: "token id_token",
            scope: window.Auth0Config.Auth0Scope,
        });
    }
})();
