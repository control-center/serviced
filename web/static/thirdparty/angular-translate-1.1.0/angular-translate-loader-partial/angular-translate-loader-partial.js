angular.module('pascalprecht.translate').provider('$translatePartialLoader', [function () {
    function Part(name) {
      this.name = name;
      this.isActive = true;
      this.tables = {};
    }
    Part.prototype.parseUrl = function (urlTemplate, targetLang) {
      return urlTemplate.replace('{part}', this.name).replace('{lang}', targetLang);
    };
    Part.prototype.getTable = function (lang, $q, $http, urlTemplate, errorHandler) {
      var deferred = $q.defer();
      if (!this.tables.hasOwnProperty(lang)) {
        var self = this;
        $http({
          method: 'GET',
          url: this.parseUrl(urlTemplate, lang)
        }).success(function (data) {
          self.tables[lang] = data;
          deferred.resolve(data);
        }).error(function () {
          if (errorHandler !== undefined) {
            errorHandler(self.name, lang).then(function (data) {
              self.tables[lang] = data;
              deferred.resolve(data);
            }, function () {
              deferred.reject(self.name);
            });
          } else
            deferred.reject(self.name);
        });
      } else
        deferred.resolve(this.tables[lang]);
      return deferred.promise;
    };
    var parts = {};
    function hasPart(name) {
      return parts.hasOwnProperty(name);
    }
    function isStringValid(str) {
      return angular.isString(str) && str !== '';
    }
    function isPartAvailable(name) {
      if (!isStringValid(name)) {
        throw new TypeError('Invalid type of a first argument, a non-empty string expected.');
      }
      return hasPart(name) && parts[name].isActive;
    }
    function deepExtend(dst, src) {
      for (var property in src) {
        if (src[property] && src[property].constructor && src[property].constructor === Object) {
          dst[property] = dst[property] || {};
          arguments.callee(dst[property], src[property]);
        } else {
          dst[property] = src[property];
        }
      }
      return dst;
    }
    this.addPart = function (name) {
      if (!isStringValid(name)) {
        throw new TypeError('Invalid type of a first argument, a non-empty string expected.');
      }
      if (!hasPart(name)) {
        parts[name] = new Part(name);
      }
      return this;
    };
    this.deletePart = function (name) {
      if (!isStringValid(name)) {
        throw new TypeError('Invalid type of a first argument, a non-empty string expected.');
      }
      delete parts[name];
      return this;
    };
    this.isPartAvailable = isPartAvailable;
    this.$get = [
      '$rootScope',
      '$injector',
      '$q',
      '$http',
      function ($rootScope, $injector, $q, $http) {
        var service = function (options) {
          if (!isStringValid(options.key)) {
            throw new TypeError('Unable to load data, a key is not a non-empty string.');
          }
          if (!isStringValid(options.urlTemplate)) {
            throw new TypeError('Unable to load data, a urlTemplate is not a non-empty string.');
          }
          var errorHandler = options.loadFailureHandler;
          if (errorHandler !== undefined) {
            if (!angular.isString(errorHandler)) {
              throw new Error('Unable to load data, a loadFailureHandler is not a string.');
            } else
              errorHandler = $injector.get(errorHandler);
          }
          var loaders = [], tables = [], deferred = $q.defer();
          function addTablePart(table) {
            tables.push(table);
          }
          for (var part in parts) {
            if (hasPart(part) && parts[part].isActive) {
              loaders.push(parts[part].getTable(options.key, $q, $http, options.urlTemplate, errorHandler).then(addTablePart));
            }
          }
          if (loaders.length) {
            $q.all(loaders).then(function () {
              var table = {};
              for (var i = 0; i < tables.length; i++) {
                deepExtend(table, tables[i]);
              }
              deferred.resolve(table);
            }, function () {
              deferred.reject(options.key);
            });
          } else {
            deferred.resolve({});
          }
          return deferred.promise;
        };
        service.addPart = function (name) {
          if (!isStringValid(name)) {
            throw new TypeError('Invalid type of a first argument, a non-empty string expected.');
          }
          if (!hasPart(name)) {
            parts[name] = new Part(name);
            $rootScope.$broadcast('$translatePartialLoaderStructureChanged', name);
          } else if (!parts[name].isActive) {
            parts[name].isActive = true;
            $rootScope.$broadcast('$translatePartialLoaderStructureChanged', name);
          }
          return service;
        };
        service.deletePart = function (name, removeData) {
          if (!isStringValid(name)) {
            throw new TypeError('Invalid type of a first argument, a non-empty string expected.');
          }
          if (removeData === undefined) {
            removeData = false;
          } else if (typeof removeData !== 'boolean') {
            throw new TypeError('Invalid type of a second argument, a boolean expected.');
          }
          if (hasPart(name)) {
            var wasActive = parts[name].isActive;
            if (removeData) {
              delete parts[name];
            } else {
              parts[name].isActive = false;
            }
            if (wasActive) {
              $rootScope.$broadcast('$translatePartialLoaderStructureChanged', name);
            }
          }
          return service;
        };
        service.isPartAvailable = isPartAvailable;
        return service;
      }
    ];
  }]);