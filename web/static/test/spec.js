/* tests */
describe('EntryControl', function() {
    var $scope = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($rootScope, $controller) {
        $scope = $rootScope.$new();
        ctrl = $controller('EntryControl', { $scope: $scope });
    }));

    it('Sets 3 main links', function() {
        expect($scope.mainlinks.length).toEqual(3);
    });

    it('Creates links that contain url and label', function() {
        for(var i=0; i < $scope.mainlinks.length; i++) {
            expect($scope.mainlinks[i].url).toMatch(/^#\/.+/);
            expect($scope.mainlinks[i].label).not.toBeUndefined();
        }
    });

});

describe('LoginControl', function() {
    var $scope = null;
    var $httpBackend = null;
    var $location = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        $httpBackend = $injector.get('$httpBackend');
        ctrl = $controller('LoginControl', { $scope: $scope });
    }));

    afterEach(function() {
        $httpBackend.verifyNoOutstandingExpectation();
        $httpBackend.verifyNoOutstandingRequest();
    });
 
    it('Sets some labels', function() {
        expect($scope.brand_label).not.toBeUndefined();
        expect($scope.login_button_text).not.toBeUndefined();
    });

    it('Sets path on successful login', function() {
        $httpBackend.when('POST', '/login').respond({Detail: 'SuccessfulPost'});
        $scope.login();
        $httpBackend.flush();
        expect($location.path()).toBe('/entry');
    });

    it('Does not change path on failed login', function() {
        $location.path('/login');
        $httpBackend.when('POST', '/login').respond(401, {Detail: 'Bad Login'});
        $scope.login();
        $httpBackend.flush();
        expect($location.path()).toBe('/login');
    });

});

describe('WizardControl', function() {
    var $scope = null;
    var $location = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        ctrl = $controller('WizardControl', { 
            $scope: $scope, 
            resourcesService: fake_resources_service(),
            wizardService: fake_wizard_service()
        });
    }));

    it('Provides a \'next\' function that checks form validity', function() {
        // Form provided with validity = true
        $location.path('/before');
        $scope.next({ $valid: true });
        expect($location.path()).toBe('/fake_next');

        // Form provided with validity = false
        $location.path('/before');
        $scope.next({ $valid: false});
        expect($location.path()).toBe('/before');

        // No form provided - should assume valid
        $location.path('/before');
        $scope.next();
        expect($location.path()).toBe('/fake_next');
    });

    it('Provides a \'cancel\' function that delegates to wizardService', function() {
        $location.path('/before');
        $scope.cancel();
        expect($location.path()).toBe('/canceled');
    });
});

describe('ConfigurationControl', function() {
    var $scope = null;
    var $location = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        ctrl = $controller('ConfigurationControl', { 
            $scope: $scope, 
            authService: fake_auth_service(),
            servicesService: fake_services_service()
        });
    }));

    it('Provides a breadcrumb', function() {
        // Has a breadcrumb
        expect($scope.breadcrumbs).not.toBeUndefined();
        // Is an array of size 1 (top level)
        expect($scope.breadcrumbs.length).toBe(1);
    });

    it('Creates a services table', function() {
        expect_table($scope.services);
    });

    it('Provides a \'click_service\' function', function() {
        expect($scope.click_service).not.toBeUndefined();
        $location.path('/before');
        // Pass a fake serviceId to click_service function
        $scope.click_service('abcdef'); 
        // Location should change and end with fake serviceId
        expect($location.path()).toMatch(/abcdef$/); 
    });
});

describe('ServiceControl', function() {
    var $scope = null;
    var $location = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        ctrl = $controller('ServiceControl', { 
            $scope: $scope, 
            $routeParams: { serviceId: 'fakeId1' },
            authService: fake_auth_service(),
            servicesService: fake_services_service()
        });
    }));

    it('Provides a breadcrumb', function() {
        // Has a breadcrumb
        expect($scope.breadcrumbs).not.toBeUndefined();
        // Is an array of size 2
        expect($scope.breadcrumbs.length).toBe(2);
    });

    it('Provides service detail headers', function() {
        expect($scope.services).not.toBeUndefined();
        expect($scope.services.headers).not.toBeUndefined();
        expect($scope.services.headers.length).not.toBeUndefined();
    });

    it('Sets the current service based on scope.params', function() {
        // This depends on 'serviceId' being set in $routeParams
        // and matching an id from $scope.services.data
        expect($scope.services).not.toBeUndefined();
        expect($scope.services.current).not.toBeUndefined();
    });
});

describe('ActionControl', function() {
    var $scope = null;
    var $location = null;
    var resourcesService = null;
    var servicesService = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        resourcesService = fake_resources_service();
        servicesService = fake_services_service();
        ctrl = $controller('ActionControl', { 
            $scope: $scope, 
            $routeParams: { poolId: 'pool123', hostId: 'host123', serviceId: 'service234' },
            servicesService: servicesService,
            resourcesService: resourcesService,
        });
        $scope.pools = {};
        $scope.hosts = {};
        $scope.services = {};
    }));
    
    it('Creates a stub for new pools', function() {
        expect($scope.newPool).not.toBeUndefined();
        expect($scope.newPool.ParentId).toBe('pool123');
    });

    it('Crates a stub for new hosts', function() {
        expect($scope.newHost).not.toBeUndefined();
        expect($scope.newHost.PoolId).toBe('pool123');
    });

    it('Creates a stub for new services', function() {
        expect($scope.newService).not.toBeUndefined();
    });

    it('Provides an \'add_pool\' function', function() {
        // add_pool method must not be called unless $scope.pools exists
        expect($scope.add_pool).not.toBeUndefined();
        $scope.newPool.Id = 'aoeu';
        $scope.newPool.ParentId = 'fakeparent';
        spyOn(resourcesService, 'add_pool');
        $scope.add_pool();
        expect(resourcesService.add_pool).toHaveBeenCalled();
        var addedPool = resourcesService.add_pool.mostRecentCall.args[0];
        // Expect the values we set earlier
        expect(addedPool).toEqual({Id: 'aoeu', ParentId: 'fakeparent'});
        // Expect newPool to be a new object with parentId from routeParams
        expect($scope.newPool).toEqual({ParentId: 'pool123'});
    });

    it('Provides an \'add_host\' function', function() {
        // add_host method must not be called unless $scope.hosts exists
        expect($scope.add_host).not.toBeUndefined();
        $scope.newHost.Id = 'aoeu';
        $scope.newHost.PoolId = 'fakeparent';
        spyOn(resourcesService, 'add_host');
        $scope.add_host();
        expect(resourcesService.add_host).toHaveBeenCalled();
        var addedHost = resourcesService.add_host.mostRecentCall.args[0];
        // Expect the values we set earlier
        expect(addedHost).toEqual({Id: 'aoeu', PoolId: 'fakeparent'});
        // Expect addedHost to be a new object with pool ID from routeParams
        expect($scope.newHost).toEqual({PoolId: 'pool123'});
    });

    it('Provides an \'add_service\' function', function() {
        // add_service method must not be called unless $scope.pools exists
        expect($scope.add_service).not.toBeUndefined();
        $scope.newService.Id = 'aoeu';
        $scope.newService.PoolId = 'fakeparent';
        spyOn(servicesService, 'add_service');
        $scope.add_service();
        expect(servicesService.add_service).toHaveBeenCalled();
        var addedService = servicesService.add_service.mostRecentCall.args[0];
        // Expect the values we set earlier
        expect(addedService).toEqual({Id: 'aoeu', PoolId: 'fakeparent'});
        // Expect newService to be an empty object
        expect($scope.newService).toEqual({});
    });

    it('Provides a \'remove_pool\' function', function() {
        spyOn(resourcesService, 'remove_pool');
        $scope.remove_pool();
        expect(resourcesService.remove_pool).toHaveBeenCalled();
        var remPoolId = resourcesService.remove_pool.mostRecentCall.args[0];
        // Pool ID comes from routeParams
        expect(remPoolId).toBe('pool123');
        // Below should be tested, but currently depends on modal dialog being hidden
        //expect($location.path()).toBe('/resources');
    });

    it('Provides a \'remove_host\' function', function() {
        spyOn(resourcesService, 'remove_host');
        $scope.remove_host();
        expect(resourcesService.remove_host).toHaveBeenCalled();
        var remHostId = resourcesService.remove_host.mostRecentCall.args[0];
        // Host ID comes from routeParams
        expect(remHostId).toBe('host123');
        // Below should be tested, but currently depends on modal dialog being hidden
        //expect($location.path()).toBe('/pools/pool123');
    });

    it('Provides a \'remove_service\' function', function() {
        spyOn(servicesService, 'remove_service');
        $scope.remove_service();
        expect(servicesService.remove_service).toHaveBeenCalled();
        // service Id comes from routeParams
        expect(servicesService.remove_service.mostRecentCall.args[0]).toBe('service234');
        // expect($location.path()).toBe('/services');7
    });

    it('Provides an \'edit_pool\' function', function() {
        // Populate $scope.pools.current. Normally this is done by the main view controller
        refreshPools($scope, resourcesService, true);
        // edit_pool method must not be called unless $scope.pools.current exists
        expect($scope.pools.current).not.toBeUndefined();
        spyOn(resourcesService, 'update_pool');
        $scope.pools.current.Name = 'editedName';
        $scope.edit_pool();

        expect(resourcesService.update_pool).toHaveBeenCalled();
        expect(resourcesService.update_pool.mostRecentCall.args[0]).toBe('pool123');
        expect(resourcesService.update_pool.mostRecentCall.args[1]).toEqual($scope.pools.current);
    });

    it('Provides an \'edit_host\' function', function() {
        // Populate $scope.hosts.current. Normally this is done by the main view controller
        refreshHosts($scope, resourcesService, true, true);
        // edit_host method must not be called unless $scope.hosts.current exists
        expect($scope.hosts.current).not.toBeUndefined();
        spyOn(resourcesService, 'update_host');
        $scope.hosts.current.Name = 'editedName';
        $scope.edit_host();

        expect(resourcesService.update_host).toHaveBeenCalled();
        expect(resourcesService.update_host.mostRecentCall.args[0]).toBe('host123');
        expect(resourcesService.update_host.mostRecentCall.args[1]).toEqual($scope.hosts.current);
    });

    it('Provides an \'edit_service\' function', function() {
        // Populate $scope.hosts.current. Normally this is done by the main view controller
        refreshServices($scope, servicesService, true);
        // edit_service method must not be called unless $scope.services.current exists
        expect($scope.services.current).not.toBeUndefined();
        spyOn(servicesService, 'update_service');
        $scope.edit_service();

        expect(servicesService.update_service).toHaveBeenCalled();
        // service Id comes from routeParams
        expect(servicesService.update_service.mostRecentCall.args[0]).toBe('service234');
        expect(servicesService.update_service.mostRecentCall.args[1]).toEqual($scope.services.current);
    });
});

describe('ResourcesControl', function() {
    var $scope = null;
    var $location = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        ctrl = $controller('ResourcesControl', { 
            $scope: $scope, 
            authService: fake_auth_service(),
            resourcesService: fake_resources_service()
        });
    }));

    it('Provides a breadcrumb', function() {
        // Has a breadcrumb
        expect($scope.breadcrumbs).not.toBeUndefined();
        // Is an array of size 1 (top level)
        expect($scope.breadcrumbs.length).toBe(1);
    });

    it('Creates a pools table', function() {
        expect_table($scope.pools);
    });

    it('Populates hosts data', function() {
        expect($scope.hosts.data).not.toBeUndefined();
    });

    it('Provides a \'click_pool\' function', function() {
        $location.path('/before');
        $scope.click_pool('1234');
        expect($location.path()).toBe('/pools/1234');
    });
});

describe('PoolControl', function() {
    var $scope = null;
    var $location = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        ctrl = $controller('PoolControl', { 
            $scope: $scope,
            $routeParams: { poolId: 'pool123' },
            authService: fake_auth_service(),
            resourcesService: fake_resources_service(),
        });
    }));

    it('Provides a breadcrumb', function() {
        // Has a breadcrumb
        expect($scope.breadcrumbs).not.toBeUndefined();
        // Is an array of size 2
        expect($scope.breadcrumbs.length).toBe(2);
    });

    it('Provides a hosts table', function() {
        expect_table($scope.hosts);
    });

    it('Provides pool detail headers', function() {
        expect($scope.pools).not.toBeUndefined();
        expect($scope.pools.headers).not.toBeUndefined();
        expect($scope.pools.headers.length).not.toBeUndefined();
        expect($scope.pools.current).not.toBeUndefined();
    });
    
    it('Provides a \'click_host\' function', function() {
        $location.path('/before');
        $scope.click_host('def1234', 'abc321');
        expect($location.path()).toBe('/pools/def1234/hosts/abc321');
    });

});

describe('HostControl', function() {
    var $scope = null;
    var $location = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        ctrl = $controller('HostControl', { 
            $scope: $scope,
            $routeParams: { poolId: 'pool123', hostId: 'host123' },
            authService: fake_auth_service(),
            resourcesService: fake_resources_service(),
        });
    }));

    it('Provides a breadcrumb', function() {
        // Has a breadcrumb
        expect($scope.breadcrumbs).not.toBeUndefined();
        // Is an array of size 3
        expect($scope.breadcrumbs.length).toBe(3);
    });

    it('Provides host details headers', function() {
        expect($scope.hosts).not.toBeUndefined();
        expect($scope.hosts.headers).not.toBeUndefined();
        expect($scope.hosts.headers.length).not.toBeUndefined();
        expect($scope.hosts.current).not.toBeUndefined();
    });
});

describe('NavbarControl', function() {
    var $scope = null;
    var $location = null;
    var $httpBackend = null;
    var $location = null;
    var authService = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        $httpBackend = $injector.get('$httpBackend');
        authService = fake_auth_service();
        ctrl = $controller('NavbarControl', { 
            $scope: $scope,
            authService: authService
        });
    }));

    afterEach(function() {
        $httpBackend.verifyNoOutstandingExpectation();
        $httpBackend.verifyNoOutstandingRequest();
    });

    it('Provides some navlinks', function() {
        expect($scope.navlinks).not.toBeUndefined();
        // 2 or more navlinks please.
        expect($scope.navlinks.length).toBeGreaterThan(1); 
    });

    it('Provides brand details', function() {
        expect($scope.brand).not.toBeUndefined();
        expect($scope.brand.url).not.toBeUndefined();
        expect($scope.brand.label).not.toBeUndefined();
    });

    it('Provides a \'logout\' function', function() {
        // Default for testing is to assume logged in.
        expect(authService.isLoggedIn()).toBe(true);
        $httpBackend.when('DELETE', '/login').respond({Detail: 'Logged Out'});
        $scope.logout();
        $httpBackend.flush();
        expect(authService.isLoggedIn()).toBe(false);
    });
});

describe('WizardService', function() {
    var $inj = null;
    var $location = null;
    var wizardService = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $inj = $injector;
        wizardService = $injector.get('wizardService');
        $location = $injector.get('$location');
    }));

    it('Provides a context that persists across controllers', function() {
        // Across multiple injections, we get the same service and same context
        var firstContext = wizardService.get_context();
        firstContext.some_data = 'foo';
        var secondService = $inj.get('wizardService');
        expect(secondService).toBe(wizardService);
        expect(secondService.get_context()).toBe(firstContext);
    });

    it('Provides a \'next_page\' function', function() {
        // Next from nowhere always goes to the start
        expect('/wizard/start').toEqual(wizardService.next_page('/before'));
        // start -> page1
        expect('/wizard/page1').toEqual(wizardService.next_page('/wizard/start'));
        // page1 -> page2
        expect('/wizard/page2').toEqual(wizardService.next_page('/wizard/page1'));
        // page2 -> finish
        expect('/wizard/finish').toEqual(wizardService.next_page('/wizard/page2'));
    });

    it('Provides a \'fix_location\' function', function() {
        // Set the location to the end of the flow and call fix_location.
        // We should wind up at the start.
        wizardService.get_context().done = {};
        $location.path('/wizard/finish');
        wizardService.fix_location($location);
        expect($location.path()).toBe('/wizard/start');

        // Assume the start page is done.
        wizardService.get_context().done = { '/wizard/start': true };
        $location.path('/wizard/finish');
        wizardService.fix_location($location);
        expect($location.path()).toBe('/wizard/page1');

        // Each step only checks the requirements for the previous page
        wizardService.get_context().done = { '/wizard/page1': true };
        $location.path('/wizard/finish');
        wizardService.fix_location($location);
        expect($location.path()).toBe('/wizard/page2');

        // If we don't understand the current page, do nothing
        wizardService.get_context().done = {};
        $location.path('/something/completely/different');
        wizardService.fix_location($location);
        expect($location.path()).toBe('/something/completely/different');
    });

    it('Provides a \'cancel_page\' function', function() {
        // Returns index path: /
        expect(wizardService.cancel_page()).toBe('/');
    });

});

describe('ServicesService', function() {
    var $scope = null;
    var $location = null;
    var $httpBackend = null;
    var $location = null;
    var servicesService = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        $location = $injector.get('$location');
        $httpBackend = $injector.get('$httpBackend');
        servicesService = $injector.get('servicesService');
    }));

    afterEach(function() {
        $httpBackend.verifyNoOutstandingExpectation();
        $httpBackend.verifyNoOutstandingRequest();
    });
    
    it('Can retrieve and cache service definitions', function() {
        // The first time GET is called, we have nothing cached so the first
        // parameter is ignored.
        $httpBackend.expect('GET', '/services').respond(fake_services());
        var serv = null;
        servicesService.get_services(false, function(data) {
            serv = data;
        });
        $httpBackend.flush();
        expect(serv).not.toBeNull();

        // We should have some cached data this time, so do not expect any
        // HTTP traffic.
        serv = null;
        servicesService.get_services(true, function(data) {
            serv = data;
        });
        expect(serv).not.toBeNull();
    });

    it('Can add new services', function() {
        var serv = { Id: 'test' };
        var resp = null;
        $httpBackend.expect('POST', '/services/add', serv).respond({ Detail: 'Added' });
        servicesService.add_service(serv, function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Added');
    });

    it('Can update existing services', function() {
        var serv = { Id: 'test', Name: 'Test2' };
        var resp = null;
        $httpBackend.expect('POST', '/services/test', serv).respond({ Detail: 'Edited' });
        servicesService.update_service(serv.Id, serv, function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Edited');
    });

    it('Can remove existing services', function() {
        var resp = null;
        $httpBackend.expect('DELETE', '/services/test').respond({ Detail: 'Deleted' });
        servicesService.remove_service('test', function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Deleted');
    });

});

describe('ResourcesService', function() {
    var $scope = null;
    var $location = null;
    var $httpBackend = null;
    var $location = null;
    var resourcesService = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        $location = $injector.get('$location');
        $httpBackend = $injector.get('$httpBackend');
        resourcesService = $injector.get('resourcesService');
    }));

    afterEach(function() {
        $httpBackend.verifyNoOutstandingExpectation();
        $httpBackend.verifyNoOutstandingRequest();
    });
    
    it('Can retrieve and cache resource pools', function() {
        // The first time GET is called, we have nothing cached so the first
        // parameter is ignored.
        $httpBackend.expect('GET', '/pools').respond(fake_pools());
        var pools = null;
        resourcesService.get_pools(false, function(data) {
            pools = data;
        });
        $httpBackend.flush();
        expect(pools).not.toBeNull();

        // We should have some cached data this time, so do not expect any
        // HTTP traffic.
        pools = null;
        resourcesService.get_pools(true, function(data) {
            pools = data;
        });
        expect(pools).not.toBeNull();
    });

    it('Can add new resource pools', function() {
        var pool = { Id: 'test' };
        var resp = null;
        $httpBackend.expect('POST', '/pools/add', pool).respond({ Detail: 'Added' });
        resourcesService.add_pool(pool, function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Added');
    });

    it('Can update existing resource pools', function() {
        var pool = { Id: 'test', Name: 'Test2' };
        var resp = null;
        $httpBackend.expect('POST', '/pools/test', pool).respond({ Detail: 'Edited' });
        resourcesService.update_pool(pool.Id, pool, function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Edited');
    });

    it('Can remove existing resource pools', function() {
        var resp = null;
        $httpBackend.expect('DELETE', '/pools/test').respond({ Detail: 'Deleted' });
        resourcesService.remove_pool('test', function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Deleted');
    });

    it('Can retrieve and cache hosts', function() {
        // The first time GET is called, we have nothing cached so the first
        // parameter is ignored.
        $httpBackend.expect('GET', '/hosts').respond(fake_hosts());
        var hosts = null;
        resourcesService.get_hosts(false, function(data) {
            hosts = data;
        });
        $httpBackend.flush();
        expect(hosts).not.toBeNull();

        // We should have some cached data this time, so do not expect any
        // HTTP traffic.
        hosts = null;
        resourcesService.get_hosts(true, function(data) {
            hosts = data;
        });
        expect(hosts).not.toBeNull();
    });

    it('Can add new hosts', function() {
        var host = { Id: 'test' };
        var resp = null;
        $httpBackend.expect('POST', '/hosts/add', host).respond({ Detail: 'Added' });
        resourcesService.add_host(host, function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Added');
    });

    it('Can update existing hosts', function() {
        var host = { Id: 'test', Name: 'Test2' };
        var resp = null;
        $httpBackend.expect('POST', '/hosts/test', host).respond({ Detail: 'Edited' });
        resourcesService.update_host(host.Id, host, function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Edited');
    });

    it('Can remove existing hosts', function() {
        var resp = null;
        $httpBackend.expect('DELETE', '/hosts/test').respond({ Detail: 'Deleted' });
        resourcesService.remove_host('test', function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Deleted');
    });
});

describe('AuthService', function() {
    var $inj = null;
    var $location = null;
    var $cookies = null;
    var authService = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $inj = $injector;
        authService = $injector.get('authService');
        $location = $injector.get('$location');
        $cookies = $injector.get('$cookies');
    }));

    it('Keeps logged in flag that persists across controllers', function() {
        var scope = {};
        $cookies = {};
        // Check the base case - defaults to false
        $location.path('/before');
        authService.checkLogin(scope);
        expect($location.path()).toBe('/login');
        expect(scope.loggedIn).toBeUndefined();

        // Check basic internal state
        $location.path('/before');
        authService.login(true);
        authService.checkLogin(scope);
        expect($location.path()).toBe('/before');
        expect(scope.loggedIn).toBe(true);

        // Check across injections
        var secondAuth = $inj.get('authService');
        expect(authService).toBe(secondAuth);
        scope = {};
        secondAuth.checkLogin(scope);
        expect($location.path()).toBe('/before');
        expect(scope.loggedIn).toBe(true);
    });

    it('Checks for a cookie', function() {
        var scope = {};
        $cookies['ZCPToken'] = 'something';
        $location.path('/before');
        authService.checkLogin(scope);
        expect(scope.loggedIn).toBe(true);
        expect($location.path()).toBe('/before');
    });
});


describe('refreshServices', function() {
    it('Puts services data in scope', function() {
        var scope = {};
        refreshServices(scope, fake_services_service(), false);
        expect(scope.services.data).toEqual(fake_services());
    });

    it('Sets the current service based on scope.params', function() {
        var dummy_data = fake_services();
        var scope = { params: { serviceId: dummy_data[0].Id }};
        refreshServices(scope, fake_services_service(), false);
        expect(scope.services.current).not.toBeUndefined();
        expect(scope.services.current.Name).toBe(dummy_data[0].Name);
    });

    it('Maps services by ID', function() {
        var dummy_data = fake_services();
        var scope = {};
        refreshServices(scope, fake_services_service(), false);
        expect(scope.services.mapped[dummy_data[0].Id].Name).toBe(dummy_data[0].Name);
        expect(scope.services.mapped[dummy_data[1].Id].Startup).toBe(dummy_data[1].Startup);
    });
});

describe('refreshPools', function() {
    it('Transforms mapped pools into array in scope', function() {
        var dummy_data = fake_pools();
        var dummy_data_array = map_to_array(dummy_data);
        var scope = {};
        refreshPools(scope, fake_resources_service(), false);
        expect(scope.pools.data).toEqual(map_to_array(dummy_data_array));
    });

    it('Puts pool data in scope', function() {
        var dummy_data = fake_pools();
        var scope = {};
        refreshPools(scope, fake_resources_service(), false);
        expect(scope.pools.mapped).toEqual(dummy_data);
    });

    it('Sets the current pool based on scope.params', function() {
        var dummy_data = fake_pools();
        var dummy_data_array = map_to_array(dummy_data);
        var scope = { params: { poolId: dummy_data_array[0].Id }};
        refreshPools(scope, fake_resources_service(), false);
        expect(scope.pools.current).not.toBeUndefined();
        expect(scope.pools.current.Name).toBe(dummy_data_array[0].Name);
    });
});

describe('refreshHosts', function() {
    it('Puts hosts filtered for current pool into scope', function() {
        var scope = { params: { poolId: "default" }};
        var dummy_hosts = fake_hosts_for_pool("default");
        refreshHosts(scope, fake_resources_service(), false, false);
        expect(scope.hosts.data).not.toBeUndefined();
        // Do a little transformation for easier testing
        var actual_hosts = {};
        scope.hosts.data.map(function(elem) {
            actual_hosts[elem.Id] = elem;
        });
        // Actual data should have expected number of hosts
        expect(scope.hosts.data.length).toBe(dummy_hosts.length);
        // Actual data should have expected hosts only
        for (var i=0; i < dummy_hosts.length; i++) {
            expect(actual_hosts[dummy_hosts[i].HostId]).not.toBeUndefined();
        }
    });

    it('Sets the current host based on scope.params', function() {
        var scope = { params: { poolId: "default", hostId: "def" }};
        refreshHosts(scope, fake_resources_service(), false, false);
        expect(scope.hosts.current).toEqual(fake_hosts()["def"]);
    });

    it('Puts host data into scope', function() {
        var scope = {};
        refreshHosts(scope, fake_resources_service(), false, false);
        expect(scope.hosts.mapped).toEqual(fake_hosts());
    });
});

describe('map_to_array', function() {
    it('Transforms map to a new array', function() {
        var dummy_data = { test1: 'abc', test2: { foo: 'bar' }};
        var dummy_data_array = map_to_array(dummy_data);
        expect(dummy_data_array).toEqual(['abc', {foo: 'bar'}]);
    });
});

describe('unauthorized', function() {
    it('Sets the path to /login', function() {
        var loc = { path: function(){} };
        spyOn(loc, 'path');
        unauthorized(loc);
        expect(loc.path).toHaveBeenCalled();
        expect(loc.path.mostRecentCall.args[0]).toBe('/login');
    });
});

describe('next_url', function() {
    it('Finds a link with name \'Next\'', function() {
        var result = next_url({ foo: 'bar', Links: [ 
            { Name: 'Baz', Url: '/something' }, 
            { Name: 'Next', Url: '/expected' }, 
            { Name: 'Other', Url: '/other' }
        ]});
        expect(result).toBe('/expected');
    });
});

describe('set_order', function() {
    it('Updates table.sort', function() {
        var table = {
            sort: 'foo',
            sort_icons: { foo: 'bar', baz: 'wibble' }
        };
        set_order('foo', table);
        expect(table.sort).toBe('-foo');
        set_order('foo', table);
        expect(table.sort).toBe('foo');
        set_order('bar', table);
        expect(table.sort).toBe('bar');
    });

    it('Updates table.sort_icons', function() {
        var table = {
            sort: 'foo',
            sort_icons: { foo: 'bar', baz: 'wibble' }
        };
        set_order('bar', table);
        expect(table.sort_icons['foo']).toBe('glyphicon-chevron-down');
        expect(table.sort_icons['bar']).toBe('glyphicon-chevron-up');
    });
});

describe('get_order_class', function() {
    it('Includes \'active\' for value or -value of table.sort', function() {
        var table = {
            sort: 'foo',
            sort_icons: { foo: 'bar', baz: 'wibble' }
        };
        expect(get_order_class('foo', table)).toMatch(/ active$/);
        table.sort = '-foo';
        expect(get_order_class('foo', table)).toMatch(/ active$/);
        expect(get_order_class('baz', table)).toMatch(/ wibble$/);
    });
});

describe('buildTable', function() {
    it('Returns object with sort_icons', function() {
        var headers = [ {id: 'foo'}, {id: 'bar'}, {id: 'baz'}];
        var table = buildTable('foo', headers);
        expect(table.sort).toBe('foo');
        expect(table.sort_icons).not.toBeUndefined();
        expect(table.set_order).not.toBeUndefined();
        expect(table.get_order_class).not.toBeUndefined();
    });

});

function fake_hosts_for_pool(poolId) {
    var mappedHosts = {
        "pool123": [{HostId: "host123", PoolId: "pool123"}],
        "default": [{HostId: "abc", PoolId: "default"}, {HostId: "def", PoolId: "default"}]
    };
    return mappedHosts[poolId];
}

function fake_resources_service() {
   return {
       get_pools: function(cacheOk, callback) {
           callback(fake_pools());
       },
       get_hosts: function(cacheOk, callback) {
           callback(fake_hosts());
       },
       get_hosts_for_pool: function(cacheOk, poolId, callback) {
           callback(fake_hosts_for_pool(poolId));
       },
       add_pool: function(pool, callback) {
           callback({});
       },
       add_host: function(host, callback) {
           callback({});
       },
       remove_pool: function(poolId, callback) {
           callback({});
       },
       remove_host: function(hostId, callback) {
           callback({});
       },
       update_pool: function(poolId, pool, callback) {
           callback({});
       },
       update_host: function(hostId, host, callback) {
           callback({});
       }
   };
}

function fake_services_service() {
    return {
        get_services: function(cacheOk, callback) {
            callback(fake_services());
        },
        add_service: function(service, callback) {
            callback({});
        },
        update_service: function(serviceId, service, callback) {
            callback({});
        },
        remove_service: function(serviceId, callback) {
            callback({});
        }
    };
}

function fake_auth_service() {
    var loggedIn = true;
    return {
        checkLogin: function(scope) {
            // stub does nothing
        },
        isLoggedIn: function() {
            return loggedIn;
        },
        login: function(truth) {
            loggedIn = truth;
        }
    };
}

function fake_wizard_service() {
    return {
        next_page: function(currentPath) {
            return "/fake_next";
        },
        cancel_page: function(currentPath) {
            return "/canceled";
        },
        get_context: function() {
            return {};
        },
        fix_location: function($location) {
            // do nothing
        }
    }
}

function expect_table(table) {
    expect(table).not.toBeUndefined();
    expect(table.data).not.toBeUndefined();
    expect(table.sort).not.toBeUndefined()
    expect(table.sort_icons).not.toBeUndefined();
    expect(table.get_order_class).not.toBeUndefined();
}

function fake_pools() {
    return {
        "default": {
            Id: "default",
            ParentId: "",
            CoreLimit: 0,
            MemoryLimit: 0,
            Priority: 0
        },
        "foo": { 
            Id: "foo",
            ParentId: "default",
            CoreLimit: 2,
            MemoryLimit: 1024,
            Priority: 1
        },
        "bar": {
            Id: "bar",
            ParentId: "default",
            CoreLimit: 8,
            MemoryLimit: 8192,
            Priority: 2
        },
        "pool123": {
            Id: "pool123",
            ParentId: "foo",
            CoreLimit: 1,
            MemoryLimit: 512,
            Priority: 2
        }
    };
}

function fake_hosts() {
    return {
        "abc": {
            Id: "abc",
            PoolId: "default",
            Name: "abchost",
            IpAddr: "192.168.33.12",
            Cores: 2,
            Memory: 3061190144,
            PrivateNetwork: "255.255.255.0"
        },
        "def": {
            Id: "def",
            PoolId: "default",
            Name: "defhost",
            IpAddr: "192.168.33.13",
            Cores: 1,
            Memory: 12345,
            PrivateNetwork: "255.255.255.0"
        },
        "host123": {
            Id: "host123",
            PoolId: "pool123",
            Name: "some fake host",
            IpAddr: "192.168.33.14",
            Cores: 2,
            Memory: 2048
        }
    };
}

function fake_services() {
    return [
        {
            "Id": "fakeId1",
            "Name": "mysql",
            "Startup": "/usr/libexec/mysqld",
            "Description": "Database service",
            "Instances": 0,
            "ImageId": "default",
            "PoolId": "default",
            "DesiredState": 1,
            "Endpoints": [
                {
                    "Protocol": "tcp",
                    "PortNumber": 3306,
                    "Application": "mysql",
                    "Purpose": "remote"
                }
            ]
        },
        {
            "Id": "service234",
            "Name": "zeneventd",
            "Startup": "/opt/zenoss/bin/zeneventd",
            "Description": "",
            "Instances": 0,
            "ImageId": "",
            "PoolId": "default",
            "DesiredState": 0,
            "Endpoints": null
        }
    ];

}

