/* global jasmine: true, spyOn: true, beforeEach: true, DEBUG: true, expect: true, inject: true, module: true */

describe('baseFactory', function() {
    beforeEach(module('baseFactory'));
    beforeEach(module('resourcesFactoryMock'));

    describe("Calls provided update function", function(){
        it("is defined", inject(function(baseFactory){
            expect(baseFactory).toBeDefined();
        }));
    });

});
