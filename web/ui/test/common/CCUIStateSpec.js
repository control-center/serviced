/* global jasmine: true, spyOn: true, beforeEach: true, DEBUG: true, expect: true, inject: true, module: true */

describe('CCUIState', function() {

    var $scope = null;
    var CCUIState = null;

    beforeEach(module('controlplaneTest'));
    beforeEach(module('CCUIState'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        CCUIState = $injector.get('CCUIState');
    }));

    it('Creates a userstore if the user doesnt exist', function() {
        var user = "themckrakken";
        CCUIState.get(user, "testStore");
        expect(CCUIState.store[user]).toBeDefined();
    });

    it('Modifies an existing user store, rather than overwriting it', function() {
        var user = "themckrakken";
        CCUIState.get(user, "testStore");
        var userStore = CCUIState.store[user];
        CCUIState.get(user, "testStore");
        expect(CCUIState.store[user]).toBe(userStore);
    });

    it('Returns a new store', function() {
        var user = "themckrakken";
        var storeName = "groceryStore";
        var store = CCUIState.get(user, storeName);
        expect(store).toBeDefined();
    });

    it('Passes a store by reference', function() {
        var user = "themckrakken";
        var storeName = "groceryStore";
        var store = CCUIState.get(user, storeName);
        store.jellyfish = "wiggle wiggle wiggle, yeah";
        var storeAgain = CCUIState.get(user, storeName);
        expect(storeAgain.jellyfish).toBe(store.jellyfish);
    });
});
