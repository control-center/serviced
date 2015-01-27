
describe('LanguageController', function() {
    var $scope = null;
    var $cookies = null;
    var $translate = null;
    var miscUtils = null

    beforeEach(module('controlplaneTest'));
    beforeEach(module(miscUtilsMock));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        $cookies = $injector.get('$cookies');
        $translate = $injector.get('$translate');
        miscUtils = $injector.get('miscUtils')

        var $controller = $injector.get('$controller');
        $controller('LanguageController', {
            $scope: $scope,
            $cookies: $cookies,
            $translate: $translate,
            miscUtils: miscUtils
        });
    }));

    it('Constructor should initialize $scope.name', function() {
        expect($scope.name).toEqual("language");
    });

    it('setUserLanguage() should set cookie', function() {
        var mockUserData = {}
        mockUserData.language = "pig_latin"
        $scope.user = mockUserData;

        $scope.setUserLanguage();

        expect($cookies.Language).toEqual(mockUserData.language);
        expect(miscUtils.updateLanguage).toHaveBeenCalledWith($scope, $cookies, $translate)
    });

    it('getLanguageClass() for matching language returns active state', function() {
        var mockUserData = {}
        mockUserData.language = "en_us"
        $scope.user = mockUserData;

        expect($scope.getLanguageClass('en_us')).toContain('active')
    });

    it('getLanguageClass() for non-matching language returns inactive state', function() {
        var mockUserData = {}
        mockUserData.language = "en_us"
        $scope.user = mockUserData;

        expect($scope.getLanguageClass('pig_latin')).not.toContain('active')
    });
});
