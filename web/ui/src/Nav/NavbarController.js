/* globals controlplane: true */
/* NavbarController.js
 * Controls the navbar. what else were you thinking it would do?
 */
(function(){
    "use strict";

    controlplane.controller("NavbarController", ["$scope", "$rootScope", "$cookies", "$location", "$route", "$translate", "$notification", "authService", "resourcesFactory", "$modalService", "miscUtils",
    function($scope, $rootScope, $cookies, $location, $route, $translate, $notification, authService, resourcesFactory, $modalService, utils){
        $scope.name = 'navbar';
        $scope.brand = { url: '#/apps', label: 'brand_cp' };

        $rootScope.messages = $notification.getMessages();
        $scope.$on('messageUpdate', function(){
            $rootScope.messages = $notification.getMessages();
            if(!$scope.$$phase) {
                $scope.$apply();
            }
        });
        $rootScope.markRead = function(message){
            $notification.markRead(message);
        };
        $rootScope.clearMessages = function(){
            $notification.clearAll();
        };

        $scope.navlinks = [
            { url: '#/apps', label: 'nav_apps', sublinks: [ '#/services/', '#/servicesmap' ] },
            { url: '#/pools', label: 'nav_pools', sublinks: [ '#/pools/' ] },
            { url: '#/hosts', label: 'nav_hosts', sublinks: [ '#/hosts/', '#/hostsmap' ] },
            { url: '#/logs', label: 'nav_logs', sublinks: [] },
            { url: '#/backuprestore', label: 'nav_backuprestore', sublinks: [] }
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

        $scope.subNavClick = function(crumb){
            // if parent scope has defined a handler
            // for sub nav click, use it
            if($scope.$parent && $scope.$parent.subNavClick){
                $scope.$parent.subNavClick(crumb);
            } else {
                $location.path(crumb.url);
            }
        };

        // watch parent for new breadcrumbs
        $scope.$watch("$parent.breadcrumbs", function(){
            $scope.breadcrumbs = $scope.$parent.breadcrumbs;
        });

        // Create a logout function
        $scope.logout = function() {
            authService.logout();
        };

        $scope.modalUserDetails = function() {
            $modalService.create({
                templateUrl: "user-details.html",
                model: $scope,
                title: "title_user_details",
                bigModal: true
            });
        };

        $scope.modalAbout = function() {
            resourcesFactory.getVersion().success(function(data){
                $scope.version = data;
            });

            $modalService.create({
                templateUrl: "about.html",
                model: $scope,
                title: "title_about"
            });
        };

        utils.updateLanguage($scope, $cookies, $translate);

        var helpMap = {
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

        $scope.cookies = $cookies;

    }]);
})();
