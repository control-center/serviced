
describe('AppsController', function() {
  var $rootScope = null,
  $scope = null,
  $routeParams = null,
  $location = null,
  $notification = null,
  resourcesFactory = null,
  authService = null,
  $modalService = null,
  $translate = null,
  $timeout = null,
  $cookies = null,
  miscUtils = null,
  ngTableParams = null,
  $filter = null,
  Service = null,
  InternalService = null,
  $q = null;


  beforeEach(module('controlplaneTest'));
  beforeEach(module(resourcesFactoryMock));

  beforeEach(module('$routeParams'));

  beforeEach(inject(function($injector) {
    $scope = $injector.get('$rootScope').$new();
    $routeParams = $injector.get('$routeParams');
    $location = $injector.get('$location');
    $notification = $injector.get('$notification');
    resourcesFactory = $injector.get('resourcesFactory');
    authService = $injector.get('authService');
    $modalService = $injector.get('$modalService');
    $translate = $injector.get('$translate');
    $timeout = $injector.get('$timeout');
    $cookies = $injector.get('$cookies');
    miscUtils = $injector.get('miscUtils');
    ngTableParams = $injector.get('ngTableParams');
    $filter = $injector.get('$filter');
    Service = $injector.get('Service');
    InternalService = $injector.get('InternalService');
    $q = $injector.get('$q');

    var $controller = $injector.get('$controller');
    var ctrl = $controller('AppsController', {
      $scope: $scope,
      $routeParams: $routeParams,
      $location: $location,
      $notification: $notification,
      resourcesFactory: resourcesFactory,
      authService: authService,
      $modalService: $modalService,
      $translate: $translate,
      $timeout: $timeout,
      $cookies: $cookies,
      miscUtils: miscUtils,
      ngTableParams: ngTableParams,
      $filter: $filter,
      Service: Service,
      InternalService: InternalService,
      $q: $q
    });
  }));

  it('has scope defined', function () {
    expect($scope).toBeDefined();
  });

  it('has templates defined', function () {
    expect($scope.templates).toBeDefined();
  });

  it('gets storage when fetching pool usage', function () {
    expect(resourcesFactory.getStorage).toHaveBeenCalled();
  });
});
