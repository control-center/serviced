/* jshint ignore: start */

// Define one global that can be reused for different test cases.
var controlplane = angular.module('controlplaneTest', ['ngMock', 'ngCookies', 'pascalprecht.translate']);

// adds success and error functions
// to regular promise ala $http
function httpify(deferred){
    deferred.promise.success = function(fn){
        deferred.promise.then(fn);
        return deferred.promise;
    };
    deferred.promise.error = function(fn){
        deferred.promise.then(null, fn);
        return deferred.promise;
    };
    return deferred;
}

// polyfill bind for PhantomJS :(
if (!Function.prototype.bind) {
  Function.prototype.bind = function(oThis) {
    if (typeof this !== 'function') {
      // closest thing possible to the ECMAScript 5
      // internal IsCallable function
      throw new TypeError('Function.prototype.bind - what is trying to be bound is not callable');
    }

    var aArgs   = Array.prototype.slice.call(arguments, 1),
        fToBind = this,
        fNOP    = function() {},
        fBound  = function() {
          return fToBind.apply(this instanceof fNOP
                 ? this
                 : oThis,
                 aArgs.concat(Array.prototype.slice.call(arguments)));
        };

    fNOP.prototype = this.prototype;
    fBound.prototype = new fNOP();

    return fBound;
  };
}

// polyfill for PhantomJS
if (!Array.prototype.find) {
  Array.prototype.find = function(predicate) {
    if (this === null) {
      throw new TypeError('Array.prototype.find called on null or undefined');
    }
    if (typeof predicate !== 'function') {
      throw new TypeError('predicate must be a function');
    }
    var list = Object(this);
    var length = list.length >>> 0;
    var thisArg = arguments[1];
    var value;

    for (var i = 0; i < length; i++) {
      value = list[i];
      if (predicate.call(thisArg, value, i, list)) {
        return value;
      }
    }
    return undefined;
  };
}
