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
    var PAGE_SIZE = 15; // TODO: pull from config file

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
                                <li class="entry"><strong>{{${tableID}.resultsLength}}</strong>
                                    Result{{ ${tableID}.resultsLength !== 1 ? "s" : ""  }}
                                </li>
                            </ul>
                        </td>
                    </tr></tfoot>
                `);

                // mark this guy as an ng-table
                table.attr("ng-table", tableID);
                table.attr("template-pagination", "/static/partials/jellyPager.html");

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
                    config().pgsize = PAGE_SIZE;

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

                    getData = function ($defer, params) {
                        var allItems = data(),
                            totalItemCount = 0,
                            sortedItems = [],
                            tableEntries = [];

                        if (angular.isUndefined(allItems)) {

                            // show loading animation and hide no-data message
                            $scope[tableID].loading = true;
                            toggleNoData(false);

                        } else {

                            $scope[tableID].loading = false;

                            if (angular.isObject(allItems) && !angular.isArray(allItems)) {
                                // make allItems an array if necessary
                                allItems = utils.mapToArr(allItems);
                            } else if (allItems === null) {
                                allItems = [];
                            }

                            totalItemCount = allItems.length;
                            // if no results show no data message
                            toggleNoData(!totalItemCount);

                            if (config().getData) {
                                // call overriden getData if available (eg services)
                                sortedItems = config().getData(allItems, params);
                            } else {
                                // use default getData (eg pools hosts)
                                sortedItems = params.sorting() ?
                                    orderBy(allItems, params.orderBy())
                                    : allItems;
                            }

                            // pagination
                            if (config().disablePagination) {
                                // supress pagination
                                tableEntries = sortedItems;
                            } else {
                                // slice sorted results array for current page
                                var lower = (params.page() - 1) * config().pgsize;
                                var upper = Math.min(lower + config().pgsize, totalItemCount);
                                tableEntries = sortedItems.slice(lower, upper);

                                if (totalItemCount > config().pgsize) {
                                    table.addClass("has-pagination");
                                    // ngtable pagination requires total item count
                                    params.total(totalItemCount);
                                } else {
                                    table.removeClass("has-pagination");
                                }
                            }
                        }

                        $scope[tableID].resultsLength = totalItemCount;
                        $scope[tableID].lastUpdate = moment.utc().tz(timezone);
                        $defer.resolve(tableEntries);
                    };

                    // setup config for ngtable
                    pageConfig = {
                        // count: hide pagination when total result count less than this number
                        count: config().pgsize,
                        sorting: config().sorting
                    };
                    dataConfig = {
                        // counts: dynamic items-per-page widget. empty array will supress.
                        counts: config().counts,
                        // pager:  dynamic items-per-page widget.
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
