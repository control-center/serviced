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
 
    it('Should set some labels', function() {
        expect($scope.brand_label).not.toBeUndefined();
        expect($scope.login_button_text).not.toBeUndefined();
    });

    it('Should set path on successful login', function() {
        $httpBackend.when('POST', '/login').respond({Detail: 'SuccessfulPost'});
        $scope.login();
        $httpBackend.flush();
        expect($location.path()).toBe('/entry');
    });

    it('Should stay put on failed login', function() {
        $httpBackend.when('POST', '/login').respond(401, {Detail: 'Bad Login'});
        $scope.login();
        $httpBackend.flush();
        expect($location.path()).not.toBe('/entry');
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

    it('Should provide a \'next\' function that checks form validity', function() {
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

    it('Should provide a \'cancel\' function that delegates to wizardService', function() {
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

    it('Should provide a breadcrumb', function() {
        // Has a breadcrumb
        expect($scope.breadcrumbs).not.toBeUndefined();
        // Is an array of size 1 (top level)
        expect($scope.breadcrumbs.length).toBe(1);
    });

    it('Should create a services table', function() {
        expect_table($scope.services);
    });

    it('Should provide a \'click_service\' function', function() {
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

    it('Should provide a breadcrumb', function() {
        // Has a breadcrumb
        expect($scope.breadcrumbs).not.toBeUndefined();
        // Is an array of size 2
        expect($scope.breadcrumbs.length).toBe(2);
    });

    it('Should provide service detail headers', function() {
        expect($scope.services).not.toBeUndefined();
        expect($scope.services.headers).not.toBeUndefined();
        expect($scope.services.headers.length).not.toBeUndefined();
    });

    it('Should provide current service', function() {
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
    
    it('Should create a stub for new pools', function() {
        expect($scope.newPool).not.toBeUndefined();
        expect($scope.newPool.ParentId).toBe('pool123');
    });

    it('Should create a stub for new hosts', function() {
        expect($scope.newHost).not.toBeUndefined();
        expect($scope.newHost.PoolId).toBe('pool123');
    });

    it('Should create a stub for new services', function() {
        expect($scope.newService).not.toBeUndefined();
    });

    it('Should provide an \'add_pool\' function', function() {
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

    it('Should provide an \'add_host\' function', function() {
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

    it('Should provide an \'add_service\' function', function() {
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

    it('Should provide a \'remove_pool\' function', function() {
        spyOn(resourcesService, 'remove_pool');
        $scope.remove_pool();
        expect(resourcesService.remove_pool).toHaveBeenCalled();
        var remPoolId = resourcesService.remove_pool.mostRecentCall.args[0];
        // Pool ID comes from routeParams
        expect(remPoolId).toBe('pool123');
        // Below should be tested, but currently depends on modal dialog being hidden
        //expect($location.path()).toBe('/resources');
    });

    it('Should provide a \'remove_host\' function', function() {
        spyOn(resourcesService, 'remove_host');
        $scope.remove_host();
        expect(resourcesService.remove_host).toHaveBeenCalled();
        var remHostId = resourcesService.remove_host.mostRecentCall.args[0];
        // Host ID comes from routeParams
        expect(remHostId).toBe('host123');
        // Below should be tested, but currently depends on modal dialog being hidden
        //expect($location.path()).toBe('/pools/pool123');
    });

    it('Should provide a \'remove_service\' function', function() {
        spyOn(servicesService, 'remove_service');
        $scope.remove_service();
        expect(servicesService.remove_service).toHaveBeenCalled();
        // service Id comes from routeParams
        expect(servicesService.remove_service.mostRecentCall.args[0]).toBe('service234');
        // expect($location.path()).toBe('/services');7
    });

    it('Should provide an \'edit_pool\' function', function() {
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

    it('Should provide an \'edit_host\' function', function() {
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

    it('Should provide an \'edit_service\' function', function() {
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

    it('Should provide a breadcrumb', function() {
        // Has a breadcrumb
        expect($scope.breadcrumbs).not.toBeUndefined();
        // Is an array of size 1 (top level)
        expect($scope.breadcrumbs.length).toBe(1);
    });

    it('Should create a pools table', function() {
        expect_table($scope.pools);
    });

    it('Should populate hosts data', function() {
        expect($scope.hosts.data).not.toBeUndefined();
    });

    it('Should provide a \'click_pool\' function', function() {
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

    it('Should provide a breadcrumb', function() {
        // Has a breadcrumb
        expect($scope.breadcrumbs).not.toBeUndefined();
        // Is an array of size 2
        expect($scope.breadcrumbs.length).toBe(2);
    });

    it('Should provide a hosts table', function() {
        expect_table($scope.hosts);
    });

    it('Should provide pool detail headers', function() {
        expect($scope.pools).not.toBeUndefined();
        expect($scope.pools.headers).not.toBeUndefined();
        expect($scope.pools.headers.length).not.toBeUndefined();
        expect($scope.pools.current).not.toBeUndefined();
    });
    
    it('Should provide a \'click_host\' function', function() {
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

    it('Should provide a breadcrumb', function() {
        // Has a breadcrumb
        expect($scope.breadcrumbs).not.toBeUndefined();
        // Is an array of size 3
        expect($scope.breadcrumbs.length).toBe(3);
    });

    it('Should provide host details headers', function() {
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

    it('Should provides some navlinks', function() {
        expect($scope.navlinks).not.toBeUndefined();
        // 2 or more navlinks please.
        expect($scope.navlinks.length).toBeGreaterThan(1); 
    });

    it('Should provide brand details', function() {
        expect($scope.brand).not.toBeUndefined();
        expect($scope.brand.url).not.toBeUndefined();
        expect($scope.brand.label).not.toBeUndefined();
    });

    it('Should provide a \'logout\' function', function() {
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

    it('Should provide a context that persists across controllers', function() {
        // Across multiple injections, we get the same service and same context
        var firstContext = wizardService.get_context();
        firstContext.some_data = 'foo';
        var secondService = $inj.get('wizardService');
        expect(secondService).toBe(wizardService);
        expect(secondService.get_context()).toBe(firstContext);
    });

    it('Should provide a \'next_page\' function', function() {
        // Next from nowhere always goes to the start
        expect('/wizard/start').toEqual(wizardService.next_page('/before'));
        // start -> page1
        expect('/wizard/page1').toEqual(wizardService.next_page('/wizard/start'));
        // page1 -> page2
        expect('/wizard/page2').toEqual(wizardService.next_page('/wizard/page1'));
        // page2 -> finish
        expect('/wizard/finish').toEqual(wizardService.next_page('/wizard/page2'));
    });

    it('Should provide a \'fix_location\' function', function() {
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

    it('Should provide a \'cancel_page\' function', function() {
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
});

function fake_hosts_for_pool(poolId) {
    var mappedHosts = {
        "pool123": [{ HostId: "host123", PoolId: "pool123" }]
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

