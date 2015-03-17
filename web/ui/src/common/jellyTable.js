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
                table.after(`<div class="loader"></div>`);
                table.find("tr").last()
                    .after(`<tr class="noData"><td colspan="100%" translate>no_data</td></tr>`)
                    .after(`<tr class="loaderSpacer"><td colspan="100%">&nbsp;</td></tr>`);

                // add table status bar
                table.append(`
                    <tfoot><tr>
                        <td colspan="100%" class="statusBar">
                            <ul>
                                <li class="entry">Last Update: <strong>{{${tableID}.lastUpdate | fromNow}}</strong></li>
                                <li class="entry">Showing <strong>{{${tableID}.resultsLength}}</strong>
                                    Result{{ ${tableID}.resultsLength > 1 ? "s" : ""  }}
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

                    var $loader, $loaderSpacer, $noData,
                        getData, pageConfig, dataConfig,
                        timezone;

                    var config = utils.propGetter($scope, attrs.config);
                    var data = utils.propGetter($scope, attrs.data);

                    timezone = jstz.determine().name();

                    // TODO - errors for missing data

                    $loader = $wrap.find(".loader");
                    $loaderSpacer = $wrap.find(".loaderSpacer");
                    $noData = $wrap.find(".noData");
                    $noData.hide();

                    getData = function($defer, params) {
                        var unorderedData = data(),
                            orderedData;

                        // call overriden getData
                        if(config().getData){
                            orderedData = config().getData(unorderedData, params);

                        // use default getData
                        } else {
                            orderedData = params.sorting() ?
                                $filter('orderBy')(unorderedData, params.orderBy()) :
                                unorderedData;
                        }

                        // if no data default it to an empty array
                        if(!orderedData){
                            orderedData = [];

                        } else {
                            // if an array was returned but
                            // is empty, show no data message
                            if(!orderedData.length){
                                $noData.show();
                            } else {
                                $noData.hide();
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

                    // watch data for changes
                    // TODO - is this expensive?
                    // TODO - custom watcher?
                    $scope.$watch(attrs.data, function(){
                        $scope[tableID].reload();
                    });

                    // show/hide loading icon
                    $scope.$watch(tableID + ".settings().$loading", function(newVal, oldVal){
                        if(oldVal === newVal) {
                            return;
                        }
                        if(newVal){
                            $loader.show();
                            // TODO - use animate, implement own loading event
                            //$animate.removeClass($loader, "disappear");
                            $loaderSpacer.show();
                        } else {
                            $loader.hide();
                            // TODO - use animate, implement own loading event
                            //$animate.addClass($loader, "disappear");
                            $loaderSpacer.hide();
                        }
                    });
                };
            }
        };
    }]);
})();
