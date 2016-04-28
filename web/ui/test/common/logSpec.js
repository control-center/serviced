/* global jasmine: true, beforeEach: true, expect: true, inject: true, module: true */

describe('log', function() {
    var $log;
    var log;

    // load up actual services
    beforeEach(function(){
        module('log');
    });

    beforeEach(inject(function($injector){
        $log = $injector.get("$log");
        log = $injector.get("log");
    }));

    it("logs debug", function(){
        log.setLogLevel(log.level.debug);
        log.debug("help");
        expect($log.debug.logs[0][1]).toEqual("help");
    });
    it("logs log", function(){
        log.setLogLevel(log.level.debug);
        log.log("help");
        expect($log.log.logs[0][1]).toEqual("help");
    });
    it("logs info", function(){
        log.setLogLevel(log.level.debug);
        log.info("help");
        expect($log.info.logs[0][1]).toEqual("help");
    });
    it("logs warnings", function(){
        log.setLogLevel(log.level.debug);
        log.warn("help");
        expect($log.warn.logs[0][1]).toEqual("help");
    });
    it("logs errors", function(){
        log.setLogLevel(log.level.debug);
        log.error("help");
        expect($log.error.logs[0][1]).toEqual("help");
    });

    it("logs only log and above", function(){
        log.setLogLevel(log.level.log);
        log.debug("help");
        expect($log.assertEmpty).not.toThrow();
        log.log("help");
        expect($log.log.logs[0][1]).toEqual("help");
        log.info("help");
        expect($log.info.logs[0][1]).toEqual("help");
        log.warn("help");
        expect($log.warn.logs[0][1]).toEqual("help");
        log.error("help");
        expect($log.error.logs[0][1]).toEqual("help");
    });
    it("logs only info and above", function(){
        log.setLogLevel(log.level.info);
        log.debug("help");
        expect($log.assertEmpty).not.toThrow();
        log.log("help");
        expect($log.assertEmpty).not.toThrow();
        log.info("help");
        expect($log.info.logs[0][1]).toEqual("help");
        log.warn("help");
        expect($log.warn.logs[0][1]).toEqual("help");
        log.error("help");
        expect($log.error.logs[0][1]).toEqual("help");
    });
    it("logs only warn and above", function(){
        log.setLogLevel(log.level.warn);
        log.debug("help");
        expect($log.assertEmpty).not.toThrow();
        log.log("help");
        expect($log.assertEmpty).not.toThrow();
        log.info("help");
        expect($log.assertEmpty).not.toThrow();
        log.warn("help");
        expect($log.warn.logs[0][1]).toEqual("help");
        log.error("help");
        expect($log.error.logs[0][1]).toEqual("help");
    });
    it("logs only errors", function(){
        log.setLogLevel(log.level.error);
        log.debug("help");
        expect($log.assertEmpty).not.toThrow();
        log.log("help");
        expect($log.assertEmpty).not.toThrow();
        log.info("help");
        expect($log.assertEmpty).not.toThrow();
        log.warn("help");
        expect($log.assertEmpty).not.toThrow();
        log.error("help");
        expect($log.error.logs[0][1]).toEqual("help");
    });
});
