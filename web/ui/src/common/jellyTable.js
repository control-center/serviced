/* globals jstz: true */
/* tableDirective.js
 * Wrapper for ngTable that gives a bit more
 * control and customization
 */

/*
 *TODO
 *generate unique id thing for ng-table property? (jellyTable1)
 *
 *
 */
(function() {
    'use strict';

    var count = 0;

    angular.module('jellyTable', [])
    .directive("jellyTable", ["$interval", "ngTableParams", "$filter", "$animate", "$compile", "miscUtils",
    function($interval, NgTableParams, $filter, $animate, $compile, utils){
        return {
            restrict: "A",
            // inherit parent scope
            scope: true,
            // ensure this directive accesses the template
            // before ng-repeat and ng-table do
            priority: 1002,
            // do not continue parsing the template
            terminal: true,
            compile: function(table){

                var $wrap, tableID, fn;

                // wrap the table up real nice
                $wrap = $(`<div class="jelly-table"></div>`);
                table.after($wrap);
                $wrap.append(table);

                // unique property name for this table
                tableID = "jellyTable" + count++;

                // add loading and no data elements
                table.find("tr").last()
                    .after(`<tr class="noData"><td colspan="100%" translate>no_data</td></tr>`)
                    .after(`<tr class="loader"><td colspan="100%">&nbsp;</td></tr>`);

                // add table status bar
                table.append(`
                    <tfoot><tr>
                        <td colspan="100%" class="statusBar">
                            <ul>
                                <li class="entry">Last Update: <strong>{{${tableID}.lastUpdate | fromNow}}</strong></li>
                                <li class="entry">Showing <strong>{{${tableID}.resultsLength}}</strong>
                                    Result{{ ${tableID}.resultsLength !== 1 ? "s" : ""  }}
                                </li>
                            </ul>
                        </td>
                    </tr></tfoot>
                `);


                // mark this guy as an ng-table
                table.attr("ng-table", tableID);

                // avoid compile loop
                table.removeAttr("jelly-table");

                // enable linker to compile and bind scope
                fn = $compile(table);

                // return link function
                return function($scope, element, attrs){
                    // bind scope to html
                    fn($scope);

                    var $loader, $noData,
                        toggleLoader, toggleNoData,
                        getData, pageConfig, dataConfig,
                        timezone, orderBy;

                    var config = utils.propGetter($scope, attrs.config);
                    var data = utils.propGetter($scope, attrs.data);

                    orderBy = $filter("orderBy");

                    // setup some config defaults
                    // TODO - create a defaults object and merge
                    // TODO - create a "defaultSort" property and use
                    // it to compose the `sorting` config option
                    config().counts = config().counts || [];
                    config().watchExpression = config().watchExpression || function(){ return data(); };

                    timezone = jstz.determine().name();

                    // TODO - errors for missing data

                    $loader = $wrap.find(".loader");
                    $noData = $wrap.find(".noData");

                    toggleLoader = function(newVal, oldVal){
                        if(oldVal === newVal){
                            return;
                        }

                        // show loading spinner
                        if(newVal){
                            $loader.show();
                            $animate.removeClass($loader, "disappear");

                        // hide loading spinner
                        } else {
                            $animate.addClass($loader, "disappear")
                                .then(function(){
                                    $loader.hide();
                                });
                        }
                    };
                    toggleNoData = function(val){
                        if(val){
                            $noData.show();
                        } else {
                            $noData.hide();
                        }
                    };

                    getData = function($defer, params) {
                        var unorderedData = data(),
                            orderedData;

                        // if unorderedData is an object, convert to array
                        // NOTE: angular.isObject does not consider null to be an object
                        if(!angular.isArray(unorderedData) && angular.isObject(unorderedData)){
                            unorderedData = utils.mapToArr(unorderedData);

                        // if it's null, create empty array
                        } else if(unorderedData === null){
                            unorderedData = [];
                        }

                        // call overriden getData
                        if(config().getData){
                            orderedData = config().getData(unorderedData, params);

                        // use default getData
                        } else {
                            orderedData = params.sorting() ?
                                orderBy(unorderedData, params.orderBy()) :
                                unorderedData;
                        }

                        // if no data, show loading and default
                        // to empty array
                        if(angular.isUndefined(orderedData)){
                            $scope[tableID].loading = true;
                            toggleNoData(false);
                            orderedData = [];

                        // if data, hide loading, and check if empty
                        // array
                        } else {
                            $scope[tableID].loading = false;
                            // if the request succeded but is
                            // just empty, show no data message
                            if(!orderedData.length){
                                toggleNoData(true);

                            // otherwise, hide no data message
                            } else {
                                toggleNoData(false);
                            }
                        }

                        $scope[tableID].resultsLength = orderedData.length;
                        $scope[tableID].lastUpdate = moment.utc().tz(timezone);

                        $defer.resolve(orderedData);
                    };

                    // setup config for ngtable
                    pageConfig = {
                        sorting: config().sorting
                    };
                    dataConfig = {
                        counts: config().counts,
                        getData: getData
                    };

                    // configure ngtable
                    $scope[tableID] = new NgTableParams(pageConfig, dataConfig);
                    $scope[tableID].loading = true;
                    toggleNoData(false);

                    // watch data for changes
                    $scope.$watch(config().watchExpression, function(){
                        $scope[tableID].reload();
                    });

                    $scope.$watch(tableID + ".loading", toggleLoader);
                };
            }
        };
    }]);
})();
