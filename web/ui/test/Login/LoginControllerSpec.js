
describe('LoginControl', function() {
    var $scope = null;
    var $location = null;
    var $notification = null;
    var $translate = null;
    var authService = null;

    beforeEach(module('controlplaneTest'));
    beforeEach(module(authServiceMock));
    beforeEach(module(miscUtilsMock));
    beforeEach(module(zenNotifyMock));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        $location = $injector.get('$location');
        $notification = $injector.get('$notification');
        $translate = $injector.get('$translate');
        authService = $injector.get('authService');

        var $controller = $injector.get('$controller');
        var ctrl = $controller('LoginController', {
            $scope: $scope,
            $location: $location,
            $notification: $notification,
            $translate: $translate,
            authService: authService
        });

    }));

    it('Constructor enables login button', function() {
        expect($scope.loginButtonText).not.toBeUndefined();
        expect($scope.loginDisabled).toBeFalsy();
    });

    it('login() passes correct credentials', function() {
        $scope.username = "gumby";
        $scope.password = "pokey";

        $scope.login();

        var credentials = authService.login.calls.argsFor(0)[0];
        expect(credentials.Username).toEqual($scope.username);
        expect(credentials.Password).toEqual($scope.password);
    });

    it('Successful login() changes route', function() {

        $scope.login();

        var successCallback = authService.login.calls.argsFor(0)[1];
        successCallback();

        expect($location.path()).toEqual('/apps');
        expect($scope.loginButtonText).toEqual("log_in");
        expect($scope.loginDisabled).toBeFalsy();
    });

    it('Failed login() does not change route', function() {
        var loginRoute = "/myLoginRoute";
        $location.path(loginRoute);

        $scope.login();

        var errorCallback = authService.login.calls.argsFor(0)[2];
        errorCallback();

        expect($location.path()).toEqual(loginRoute);
        expect($scope.loginButtonText).toEqual("log_in");
        expect($scope.loginDisabled).toBeFalsy();
    });
});
