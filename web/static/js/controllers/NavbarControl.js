function NavbarControl($scope, $rootScope, $http, $cookies, $location, $route, $translate, $notification, authService) {
    $scope.name = 'navbar';
    $scope.brand = { url: '#/entry', label: 'brand_cp' };

    $rootScope.messages = $notification.getMessages();
    $scope.$on('messageUpdate', function(){
        $rootScope.messages = $notification.getMessages();
        if(!$scope.$$phase) {
            $scope.$apply();
        }
    });
    $rootScope.markRead = function(message){
        $notification.markRead(message);
    }
    $rootScope.clearMessages = function(){
        $notification.clearAll();
    }

    $scope.navlinks = [
        { url: '#/apps', label: 'nav_apps',
            sublinks: [ '#/services/', '#/servicesmap' ], target: "_self"
        },
        { url: '#/pools', label: 'nav_pools',
            sublinks: [ '#/pools/' ], target: "_self"
        },
        { url: '#/hosts', label: 'nav_hosts',
            sublinks: [ '#/hosts/', '#/hostsmap' ], target: "_self"
        },
        { url: '#/logs', label: 'nav_logs',
            sublinks: [], target: "_self"
        },
        { url: '#/backuprestore', label: 'nav_backuprestore',
            sublinks: [], target: "_self"
        }
    ];

    for (var i=0; i < $scope.navlinks.length; i++) {
        var cls = '';
        var currUrl = '#' + $location.path();
        if ($scope.navlinks[i].url === currUrl) {
            cls = 'active';
        } else {
            for (var j=0; j < $scope.navlinks[i].sublinks.length; j++) {
                if (currUrl.indexOf($scope.navlinks[i].sublinks[j]) === 0) {
                    cls = 'active';
                }
            }
        }
        $scope.navlinks[i].itemClass = cls;
    }

    // Create a logout function
    $scope.logout = function() {
        // Set internal state to logged out.
        authService.login(false);
        // Make http call to logout from server
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
    };

    $scope.modalUserDetails = function() {
        $('#userDetails').modal('show');
    };
    updateLanguage($scope, $cookies, $translate);

    var helpMap = {
        '/static/partials/main.html': 'main.html',
        '/static/partials/login.html': 'login.html',
        '/static/partials/view-subservices.html': 'subservices.html',
        '/static/partials/view-apps.html': 'apps.html',
        '/static/partials/view-hosts.html': 'hosts.html',
        '/static/partials/view-host-map.html': 'hostmap.html',
        '/static/partials/view-service-map.html': 'servicemap.html',
        '/static/partials/view-host-details.html': 'hostdetails.html',
        '/static/partials/view-devmode.html': 'devmode.html'
    };

    $scope.help = {
        url: function() {
            return '/static/help/' + $scope.user.language + '/' + helpMap[$route.current.templateUrl];
        }
    };

}
