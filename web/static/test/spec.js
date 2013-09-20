describe('EntryControl', function() {
    var $scope = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($rootScope, $controller) {
        $scope = $rootScope.$new();
        ctrl = $controller('EntryControl', { $scope: $scope });
    }));

    it('Should set 3 main links', function() {
        expect($scope.mainlinks.length).toEqual(3);
    });

    it('Should create links that contain url and label', function() {
        for(var i=0; i < $scope.mainlinks.length; i++) {
            expect($scope.mainlinks[i].url).toMatch(/^#\/.+/);
            expect($scope.mainlinks[i].label).not.toBeUndefined();
        }
    });

});

describe('LoginControl', function() {
    var $scope = null;
    var $httpBackend = null;
    var ctrl = null;

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $httpBackend = $injector.get('$httpBackend');
        $httpBackend.when('POST', '/login').respond({Detail: 'SuccessfulPost'});
        ctrl = $controller('LoginControl', { $scope: $scope });
    }));

    afterEach(function() {
        $httpBackend.verifyNoOutstandingExpectation();
        $httpBackend.verifyNoOutstandingRequest();
    });
 
    it('Should set some labels', function() {
        expect($scope.brand_label).not.toBeUndefined();
        expect($scope.login_button_text).not.toBeUndefined();
    });

    it('Should have a login function that posts', function() {
        $scope.login();
        $httpBackend.flush();
        expect($scope.extra_class).toBe('has-success');
    });
});
