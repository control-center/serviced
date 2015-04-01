/* globals DEBUG: true */

/* miscUtils.js
 * miscellaneous utils and stuff that
 * doesn't quite fit in elsewhere
 */
(function(){
    "use strict";

    angular.module("miscUtils", [])
    .factory("miscUtils", [ "$parse",
    function($parse){

        var utils = {
            /*
             * Functions for setting up grid views
             * TODO - create angular controller for grids
             */
            buildTable: function(sort, headers) {
                var sort_icons = {};
                for(var i=0; i < headers.length; i++) {
                    sort_icons[headers[i].id] = (sort === headers[i].id?
                        'glyphicon-chevron-up' : 'glyphicon-chevron-down');
                }

                return {
                    sort: sort,
                    reverse: false,
                    headers: headers,
                    sort_icons: sort_icons,
                    set_order: utils.set_order,
                    get_order_class: utils.get_order_class,
                };
            },

            set_order: function(order, table) {
                // Reset the icon for the last order
                if(DEBUG){
                    console.log('Resetting ' + table.sort + ' to down.');
                }
                table.sort_icons[table.sort] = 'glyphicon-chevron-down';

                if (table.sort === order) {
                    table.reverse = true;
                    table.sort = "-" + order;
                    table.sort_icons[table.sort] = 'glyphicon-chevron-down';
                    if(DEBUG){
                        console.log('Sorting by -' + order);
                    }
                } else {
                    table.sort = order;
                    table.reverse = false;
                    table.sort_icons[table.sort] = 'glyphicon-chevron-up';
                    if(DEBUG){
                        console.log('Sorting ' + table +' by ' + order);
                    }
                }
            },

            get_order_class: function(order, table) {
                return'glyphicon btn-link sort pull-right ' + table.sort_icons[order] +
                    ((table.sort === order || table.sort === '-' + order) ? ' active' : '');
            },


            /*
             * Helper and utility functions
             */
            // TODO - use angular $location object to make this testable
            unauthorized: function() {
                console.error('You don\'t appear to be logged in.');
                // show the login page and then refresh so we lose any incorrect state. CC-279
                window.location.href = "/#/login";
                window.location.reload();
            },

            indentClass: function(depth) {
                return 'indent' + (depth -1);
            },

            downloadFile: function(url){
                window.location = url;
            },

            getModeFromFilename: function(filename){
                var re = /(?:\.([^.]+))?$/;
                var ext = re.exec(filename)[1];
                var mode;
                switch(ext) {
                    case "conf":
                        mode="properties";
                        break;
                    case "xml":
                        mode = "xml";
                        break;
                    case "yaml":
                        mode = "yaml";
                        break;
                    case "txt":
                        mode = "plain";
                        break;
                        case "json":
                        mode = "javascript";
                        break;
                    default:
                        mode = "shell";
                        break;
                }

                return mode;
            },

            updateLanguage: function updateLanguage($scope, $cookies, $translate) {
                var ln = 'en_US';
                if ($cookies.Language === undefined) {

                } else {
                    ln = $cookies.Language;
                }
                if ($scope.user) {
                    $scope.user.language = ln;
                }
                $translate.use(ln);
            },

            capitalizeFirst: function(str){
                return str.slice(0,1).toUpperCase() + str.slice(1);
            },

            // call fn b after fn a
            after: function(a, b, context){
                return function(){
                    var results;
                    results = a.apply(context, arguments);
                    // TODO - send results to b?
                    b.call(context);
                    return results;
                };
            },

            mapToArr: function(data) {
                var arr = [];
                for (var key in data) {
                    arr.push(data[key]);
                }
                return arr;
            },


            // cache function results based on hash function.
            // NOTE: unlike regular memoize, the caching is entirely
            // based on hash function, not on arguments
            memoize: function(fn, hash){
                var cache = {};
                return function(){
                    var key = hash.apply(this, arguments),
                        val;

                    // if value isnt cached, evaluate and cache
                    if(!(key in cache)){
                        val = fn.apply(this, arguments);
                        cache[key] = val;
                    } else {
                        val = cache[key];
                    }

                    return val;
                };
            },

            needsHostAlias: function(host){
                // check is location.hostname is an IP
                var re = /\b(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b/;
                return re.test(host) || host === "localhost";
            },

            parseEngineeringNotation: function(str) {
                // Converts nK/k, nM/m, nG/g, nT/t to a number. e.g. 1K returns 1024.
                if (str === "" || str === undefined) {
                    return 0;
                }
                var prefix = parseFloat(str);
                var suffix = str.slice(prefix.toString().length);
                switch(suffix) {
                    case "K":
                    case "k":
                        prefix *= (1 << 10);
                        break;
                    case "M":
                    case "m":
                        prefix *= (1 << 20);
                        break;
                    case "G":
                    case "g":
                        prefix *= (1 << 30);
                        break;
                    case "T":
                    case "t":
                        prefix *= (1 << 40);
                        break;
                }
                return prefix;
            },

            // returns a function that will parse the
            // expression `attr` on scope object $scope
            // and return that value
            propGetter: function($scope, attr){
                var getter = $parse(attr);
                return function(){
                    return getter($scope);
                };
            }
        };

        return utils;
    }]);
})();
