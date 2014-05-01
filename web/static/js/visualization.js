/**
 * @overview Provides objects and convenience methods to construct and
 *           manipulate Zenoss visualization graphs.
 * @copyright 2013, Zenoss, Inc; All rights reserved
 */
/**
 * @namespace
 */

(function(window) {
    "use strict";

    // make sure that Array.forEach is available
    if (!('forEach' in Array.prototype)) {
        Array.prototype.forEach= function(action, that /*opt*/) {
            for (var i= 0, n= this.length; i<n; i++) {
                if (i in this) {
                    action.call(that, this[i], i, this);
                }
            }
        };
    }
    var DEFAULT_NUMBER_FORMAT = "%6.2f";

    /**
     * @namespace zenoss
     */
    var zenoss = {

        /**
         * @memberOf zenoss
         * @namespace
         * @access public
         */
        visualization : {

            /**
             * Used to enable (true) or disable (false) debug output to the
             * browser console
             *
             * @access public
             * @default false
             */
            debug : true,

            /**
             * Used to specify the base URL that is the endpoint for the Zenoss
             * metric service.
             *
             * @access public
             * @default http://localhost:8080
             */
            url : "http://localhost:8080",

            /**
             * The url path where the static javascript dependencies can be
             * found. This includes library dependencies like jquery.
             *
             * @access public
             * @default /static/performance/query
             */
            urlPath : "/static/performance/query/",

            /**
             * The url path where metrics are fetched from the server
             *
             * @access public
             * @default /api/performance/query
             */
            urlPerformance : "/api/performance/query/",

            /**
             * Determines if the legend is displayed when no data is available
             * for any plot in the grid.
             *
             * @access public
             * @default true
             */
            showLegendOnNoData : true,

            /**
             * Used for formatting the date in the legend of the chart.
             * It must be a valid moment.js date format.
             * http://momentjs.com/docs/#/parsing/string-format/
             * @access public
             * @default "MM/DD/YY hh:mm:ss a"
             */
            dateFormat: "MM/DD/YY hh:mm:ss a",
            /**
             * Used to format dates for the output display in the footer of a
             * chart.
             *
             * @param {int}
             *            unix timestamp of the date to be formated
             * @returns a string representation of the date
             * @access public
             */
            dateFormatter : function(date, timezone) {
                return moment.utc(date, "X").tz(timezone).format(zenoss.visualization.dateFormat);
            },

            /**
             * Used to generate the date/time to be displayed on a tick mark.
             * This takes into account the range of times being displayed so
             * that common data can be removed.
             *
             * @param {Date}
             *            start the start date of the time range being
             *            considered
             * @param {Date}
             *            end the end of the time range being considerd
             * @param {timestamp}
             *            ts the timestamp to be formated in ms since epoch
             * @returns string representation of the timestamp
             * @access public
             */
            tickFormat : function(start, end, ts, timezone) {
                var _start, _end, ts_seconds;

                /*
                 * Convert the strings to date instances, with the understanding
                 * that that data strings may be the one passed back from the
                 * metric service that have '-' instead of spaces
                 */
                if ($.isNumeric(start)) {
                    _start = new Date(start * 1000);
                } else {
                    _start = start;
                }

                if ($.isNumeric(end)) {
                    _end = new Date(end * 1000);
                } else {
                    _end = end;
                }

                //NOTE: Javascript timestamps are usually in milliseconds,
                // but moment.js uses seconds so we have to divide by 1000
                ts_seconds = ts / 1000;
                // Select a date/time format based on the range
                if (_start.getFullYear() === _end.getFullYear()) {
                    if (_start.getMonth() === _end.getMonth()) {
                        if (_start.getDate() === _end.getDate()) {
                            if (_start.getHours() === _end.getHours()) {
                                if (_start.getMinutes() === _end.getMinutes()) {
                                    // only show seconds
                                    return moment.utc(ts_seconds, "X").tz(timezone).format("::ss");
                                }
                                // show minutes and seconds
                                return moment.utc(ts_seconds, "X").tz(timezone).format(":mm :ss");
                            }
                            // hours, minutes and seconds
                            return moment.utc(ts_seconds, "X").tz(timezone).format("hh:mm:ssa");
                        }
                    }
                    //show the date
                    return moment.utc(ts_seconds, "X").tz(timezone).format("MM/DD-hh:mm:ssa");
                }
                // show the full date
                return moment.utc(ts_seconds, "X").tz(timezone).format(zenoss.visualization.dateFormat);
            },

            /**
             * Wrapper around the console group function. This wrapper protects
             * the client from those browsers that don't support the group
             * function.
             *
             * @access private
             */
            __group : function() {
                if (console !== undefined) {
                    if (console.group !== undefined) {
                        console.group.apply(console, arguments);
                    } else if (console.log !== undefined) {
                        console.log.apply(console, arguments);
                    }
                    // Oh well
                }
            },

            /**
             * Wrapper around the console groupCollapsed function. This wrapper
             * protects the client from those browsers that don't support this
             * function.
             *
             * @access private
             */
            __groupCollapsed : function() {
                if (console !== undefined) {
                    if (console.groupCollapsed !== undefined) {
                        console.groupCollapsed.apply(console, arguments);
                    } else if (console.log !== undefined) {
                        console.log.apply(console, arguments);
                    }
                    // Oh well
                }
            },

            /**
             * Wrapper around the console function. This wrapper protects the
             * client from those browsers that don't support this function.
             *
             * @access private
             */
            __groupEnd : function() {
                if (console !== undefined) {
                    if (console.groupEnd !== undefined) {
                        console.groupEnd.apply(console, arguments);
                    } else if (console.log !== undefined) {
                        console.log.apply(console, [ "END" ]);
                    }
                    // Oh well
                }
            },

            /**
             * Wrapper around the console function. This wrapper protects the
             * client from those browsers that don't support this function.
             *
             * @access private
             */
            __error : function() {
                if (console !== undefined) {
                    if (console.error !== undefined) {
                        console.error(arguments[0]);
                    } else if (console.log !== undefined) {
                        console.log.apply(console, arguments);
                    }
                    // If neither of those exists, oh well ....
                }
            },

            /**
             * Wrapper around the console function. This wrapper protects the
             * client from those browsers that don't support this function.
             *
             * @access private
             */
            __warn : function() {
                if (console !== undefined) {
                    if (console.warn !== undefined) {
                        console.warn(arguments[0]);
                    } else if (console.log !== undefined) {
                        console.log.apply(console, arguments);
                    }
                    // If neither of those exists, oh well ....
                }
            },

            /**
             * Wrapper around the console function. This wrapper protects the
             * client from those browsers that don't support this function.
             *
             * @access private
             */
            __log : function() {
                if (console !== undefined) {
                    if (console.log !== undefined) {
                        console.log.apply(console, arguments);
                    }
                    // Oh well
                }
            },

            /**
             * Culls the plots in a chart so that only data points with a common
             * time stamp remain.
             *
             * @param the
             *            chart that contains the plots to cull
             * @access private
             */
            __cull : function(chart) {

                var i, keys = [];
                /*
                 * If there is only one plot in the chart we are done, there is
                 * nothing to be done.
                 */
                if (chart.plots.length < 2) {
                    return;
                }

                chart.plots.forEach(function(plot) {
                    plot.values.forEach(function(v) {
                        if (keys[v.x] === undefined) {
                            keys[v.x] = 1;
                        } else {
                            keys[v.x] += 1;
                        }
                    });
                });

                // At this point, any entry in the keys array with a count of
                // chart.plots.length is a key in every plot and we can use, so
                // now
                // we walk through the plots again removing any invalid key
                chart.plots.forEach(function(plot) {
                    for (i = plot.values.length - 1; i >= 0; i -= 1) {
                        if (keys[plot.values[i].x] !== chart.plots.length) {
                            plot.values.splice(i, 1);
                        }
                    }
                });
            },

            __reduceMax : function(group) {
                return group.reduce(function(p, v) {
                    if (p.values[v.y] === undefined) {
                        p.values[v.y] = 1;
                    } else {
                        p.values[v.y] += 1;
                    }
                    p.max = Math.max(p.max, v.y);
                    return p;
                }, function(p, v) {
                    var k;
                    // need to remove the value from the values array
                    p.values[v.y] -= 1;
                    if (p.values[v.y] <= 0) {
                        delete p.values[v.y];
                        if (p.max === v.y) {
                            // pick new max, by iterating over keys
                            // finding the largest.
                            p.max = -1;
                            for (k in p.values) {
                                if (p.values.hasOwnProperty(k)) {
                                    p.max = Math.max(p.max, parseFloat(k));
                                }
                            }
                        }
                    }
                    p.total -= v.y;
                    return p;
                }, function() {
                    return {
                        values : {},
                        max : -1,
                        toString : function() {
                            return this.max;
                        }
                    };
                });
            },

            /**
             * Used to augment the div element with an error message when an
             * error is encountered while creating a chart.
             *
             * @access private
             * @param {string}
             *            name the ID of the HTML div element to augment
             * @param {object}
             *            err the error object
             * @param {string}
             *            detail the detailed error message
             */
            __showError : function(name, detail) {
                zenoss.visualization.__showMessage(name,
                    '<span class="zenerror">' + detail + '</span>');
            },

            /**
             * Shows a no data available message in the chart and hides any
             * chart elements such as the chart and the footer.
             *
             * @access private
             * @param {string}
             *            name of the div wrapper for the chart
             */
            __showNoData : function(name) {
                zenoss.visualization.__showMessage(name,
                    '<span class="nodata"></span>');
            },

            /**
             * Hides the message window
             *
             * @access private
             * @param {string}
             *            name of the div wrapper for the chart
             */
            __hideMessage : function(name) {
                $('#' + name + ' .message').css('display', 'none');
            },

            /**
             * Show the message window and hide the chart elements. The message
             * window is then populated with the given message.
             *
             * @access private
             * @param {string}
             *            name of the div wrapper for the chart
             * @param {string}
             *            html that represents the message to display.
             */
            __showMessage : function(name, message) {
                if (message) {
                    $('#' + name + ' .message').html(message);
                }
                zenoss.visualization.__hideChart(name);

                // Center the message in the div
                $('#' + name + ' .message').css('display', 'block');
                $('#' + name + ' .message span').css('position', 'relative');
                $('#' + name + ' .message span').width(
                    $('#' + name + ' .message').width()
                        - parseInt($('#' + name + ' .message span')
                        .css('margin-left'), 10)
                        - parseInt($('#' + name + ' .message span')
                        .css('margin-right'), 10));
                $('#' + name + ' .message span').css('top', '50%');
                $('#' + name + ' .message span')
                    .css(
                        'margin-top',
                        -parseInt($('#' + name + ' .message span')
                            .height(), 10) / 2);
            },

            /**
             * Hides the chart elements
             *
             * @access private
             * @param {string}
             *            name of the div wrapper of the chart
             */
            __hideChart : function(name) {
                $('#' + name + ' .zenchart').css('display', 'none');

                if (!zenoss.visualization.showLegendOnNoData) {
                    $('#' + name + ' .zenfooter').css('display', 'none');
                }
            },

            /**
             * Shows the chart elements
             *
             * @access private
             * @param {string}
             *            name of the div wrapper of the chart
             */
            __showChart : function(name) {
                zenoss.visualization.__hideMessage(name);
                $('#' + name + ' .zenchart').css('display', 'block');
                $('#' + name + ' .zenfooter').css('display', 'block');

            },

            Error : function(name, message) {
                this.name = name;
                this.message = message;
                this.toString = function() {
                    return this.name + ' : ' + this.message;
                };
            },

            /**
             * This class should not be instantiated directly unless the caller
             * really understand what is going on behind the scenes as there is
             * a lot of concurrent processing involved as many components are
             * loaded dynamically with a delayed creation or realization.
             *
             * Instead instance of this class are better created with the
             * zenoss.visualization.chart.create method.
             *
             * @access private
             * @constructor
             * @param {string}
             *            name the name of the HTML div element to augment with
             *            the chart
             * @param {object}
             *            config the values specified as the configuration will
             *            augment / override options loaded from any chart
             *            template that is specified, thus if no chart template
             *            is specified this configuration parameter can be used
             *            to specify the entire chart definition.
             */
            Chart : function(name, config) {
                this.name = name;
                this.config = config;
                this.yAxisLabel = config.yAxisLabel;
                this.div = $('#' + this.name);
                if (this.div[0] === undefined) {
                    throw new zenoss.visualization.Error('SelectorError',
                        'unknown selector specified, "' + this.name + '"');
                }

                // Build up a map of metric name to legend label.
                this.__buildPlotInfo();

                this.overlays = config.overlays || [];
                // set the format or a default
                this.format = config.format || DEFAULT_NUMBER_FORMAT;
                if ($.isNumeric(config.miny)) {
                    this.miny = config.miny;
                }
                if ($.isNumeric(config.maxy)) {
                    this.maxy = config.maxy;
                }
                this.timezone = config.timezone || jstz.determine().name();
                this.svgwrapper = document.createElement('div');
                $(this.svgwrapper).addClass('zenchart');
                $(this.div).append($(this.svgwrapper));
                this.containerSelector = '#' + name + ' .zenchart';

                this.message = document.createElement('div');
                $(this.message).addClass('message');
                $(this.message).css('display', 'none');
                $(this.div).append($(this.message));

                this.footer = document.createElement('div');
                $(this.footer).addClass('zenfooter');
                $(this.div).append($(this.footer));

                this.svg = d3.select(this.svgwrapper).append('svg');
                try {
                    this.request = this.__buildDataRequest(this.config);

                    if (zenoss.visualization.debug) {
                        zenoss.visualization
                            .__groupCollapsed('POST Request Object');
                        zenoss.visualization.__log(zenoss.visualization.url
                            + zenoss.visualization.urlPerformance);
                        zenoss.visualization.__log(this.request);
                        zenoss.visualization.__groupEnd();
                    }

                    /*
                     * Sanity Check. If the request contained no metrics to
                     * query then log this information as a warning, as it
                     * really does not make sense.
                     */
                    if (this.request.metrics === undefined) {
                        zenoss.visualization
                            .__warn('Chart configuration contains no metric sepcifications. No data will be displayed.');
                    }
                    this.update();
                } catch (x) {
                    zenoss.visualization.__error(x);
                    zenoss.visualization.__showError(this.name, x);
                }
            },

            /**
             * @namespace
             * @access public
             */
            chart : {
                /**
                 * Looks up a chart instance by the given name and, if found,
                 * updates the chart instance with the given changes. To remove
                 * an item (at the first level or the change structure) set its
                 * values to the negative '-' symbol.
                 *
                 * @param {string}
                 *            name the name of the chart to update
                 * @param {object}
                 *            changes a configuration object that holds the
                 *            changes to the chart
                 */
                update : function(name, changes) {
                    var found = zenoss.visualization.__charts[name];
                    if (found === undefined) {
                        zenoss.visualization
                            .__warn('Attempt to modify a chart, "' + name
                                + '", that does not exist.');
                        return;
                    }
                    found.update(changes);
                },

                /**
                 * Constructs a zenoss.visualization.Chart object, but first
                 * dynamically loading any chart definition required, then
                 * dynamically loading all dependencies, and finally creating
                 * the chart object. This method should be used to create a
                 * chart as opposed to calling "new" directly on the class.
                 *
                 * @param {string}
                 *            name the name of the HTML div element to augment
                 *            with the chart
                 * @param {string}
                 *            [template] the name of the chart template to load.
                 *            The chart template will be looked up as a resource
                 *            against the Zenoss metric service.
                 * @param {object}
                 *            [config] the values specified as the configuration
                 *            will augment / override options loaded from any
                 *            chart template that is specified, thus if no chart
                 *            template is specified this configuration parameter
                 *            can be used to specify the entire chart
                 *            definition.
                 * @param {callback}
                 *            [success] this callback will be called when a
                 *            zenoss.visualization.Chart object is successfully
                 *            created. The reference to the Chart object will be
                 *            passed as a parameter to the callback.
                 * @param {callback}
                 *            [fail] this callback will be called when an error
                 *            is encountered during the creation of the chart.
                 *            The error that occurred will be passed as a
                 *            parameter to the callback.
                 */
                create : function(name, arg1, arg2, success, fail) {
                    function loadChart(name, callback, onerror) {
                        var _callback = callback;
                        if (zenoss.visualization.debug) {
                            zenoss.visualization.__log('Loading chart from: '
                                + zenoss.visualization.url + '/chart/name/'
                                + name);
                        }
                        $
                            .ajax({
                                'url' : zenoss.visualization.url
                                    + '/chart/name/' + name,
                                'type' : 'GET',
                                'dataType' : 'json',
                                'contentType' : 'application/json',
                                'success' : function(data) {
                                    _callback(data);
                                },
                                'error' : function(response) {
                                    var err, detail;
                                    zenoss.visualization
                                        .__error(response.responseText);
                                    err = JSON.parse(response.responseText);
                                    detail = 'Error while attempting to fetch chart resource with the name "'
                                        + name
                                        + '", via the URL "'
                                        + zenoss.visualization.url
                                        + '/chart/name/'
                                        + name
                                        + '", the reported error was "'
                                        + err.errorSource
                                        + ':'
                                        + err.errorMessage + '"';
                                    if (onerror !== undefined) {
                                        onerror(err, detail);
                                    }
                                }
                            });
                    }

                    var config, template, result;
                    if (typeof arg1 === 'string') {
                        // A chart template name was specified, so we need to
                        // first
                        // load that template and then create the chart based on
                        // that.

                        config = arg2;
                        if (window.jQuery === undefined) {
                            zenoss.visualization
                                .__bootstrap(function() {
                                    loadChart(
                                        arg1,
                                        function(template) {
                                            var merged = new zenoss.visualization.Chart(
                                                name,
                                                zenoss.visualization
                                                    .__merge(
                                                        template,
                                                        config));
                                            zenoss.visualization.__charts[name] = merged;
                                            return merged;
                                        }, function(err, detail) {
                                            zenoss.visualization
                                                .__showError(name,
                                                    detail);
                                        });
                                });
                            return;
                        }
                        loadChart(arg1, function(template) {
                            var merged = new zenoss.visualization.Chart(name,
                                zenoss.visualization.__merge(template,
                                    config));
                            zenoss.visualization.__charts[name] = merged;
                            return merged;
                        }, function(err, detail) {
                            zenoss.visualization.__showError(name, detail);
                        });
                        return;
                    }

                    template = null;
                    config = arg1;

                    if (window.jQuery === undefined) {
                        zenoss.visualization.__bootstrap(function() {
                            var merged = new zenoss.visualization.Chart(name,
                                zenoss.visualization.__merge(template,
                                    config));
                            zenoss.visualization.__charts[name] = merged;
                            return merged;
                        });
                        return;
                    }
                    result = new zenoss.visualization.Chart(name,
                        zenoss.visualization.__merge(template, config));
                    zenoss.visualization.__charts[name] = result;
                }
            },

            /**
             * Used to track dependency loading, including the load state
             * (loaded / loading) as well as the callback that will be called
             * when a dependency load has been completed.
             *
             * @access private
             */
            __dependencies : {},

            /**
             * Used to track the charts that have been created and the names to
             * which they are associated
             *
             * @access private
             */
            __charts : {},

            /**
             * Main entry point for web pages. This method is used to first
             * bootstrap the library and then call the callback to create
             * charts. Because of the updated dependency loading capability,
             * this method is not strictly needed any more, but will be left
             * around for posterity.
             *
             * @param {callback}
             *            callback method called after all the pre-requisite
             *            JavaScript libraries are loaded.
             */
            load : function(callback) {
                zenoss.visualization.__bootstrap(callback);
            }
        }
    };

    if (typeof String.prototype.endsWith !== 'function') {
        String.prototype.endsWith = function(suffix) {
            return this.indexOf(suffix, this.length - suffix.length) !== -1;
        };
    }

    if (typeof String.prototype.startsWith !== 'function') {
        String.prototype.startsWith = function(str) {
            return this.slice(0, str.length) === str;
        };
    }

    /*
     * Symbols used during autoscaling
     */
    zenoss.visualization.__scaleSymbols = [ 'y', // 10e-24 Yecto
        'z', // 10^-21 Zepto
        'a', // 10^-18 Atto
        'f', // 10^-15 Femto
        'p', // 10^-12 Pico
        'n', // 10^-9 Nano
        'u', // 10^-6 Micro
        'm', // 10^-3 Milli
        ' ', // Base
        'k', // 10^3 Kilo
        'M', // 10^6 Mega
        'G', // 10^9 Giga
        'T', // 10^12 Tera
        'P', // 10^15 Peta
        'E', // 10^18 Exa
        'Z', // 10^21 Zetta
        'Y' // 10^24 Yotta
    ];

    /**
     * Returns the appropriate scale symbol given a scaling factor
     *
     * @access private
     * @param {number}
     *            scale factor, which is the the value which is multiplied by
     *            the scale unit and then applied to a value to get the
     *            displayed value.
     * @returns character the symbol associated widh the given scale factor
     */
    zenoss.visualization.Chart.prototype.__scaleSymbol = function(factor) {
        var ll, idx;
        ll = zenoss.visualization.__scaleSymbols.length;
        idx = factor + ((ll - 1) / 2);
        if (idx < 0 || idx >= ll) {
            return 'UKN';
        }
        return zenoss.visualization.__scaleSymbols[idx];
    };

    /**
     * Calculates a scale factor given the maximum value in the chart.
     *
     * @access private
     * @param {number}
     *            maximum value in the chart data
     * @returns number the calculated scale factor
     */
    zenoss.visualization.Chart.prototype.__calculateAutoScaleFactor = function(
        max) {
        var factor = 0, ceiling, upper, lower, unit;
        if (this.config.autoscale) {
            ceiling = this.config.autoscale.ceiling || 5;
            unit = parseInt(this.config.autoscale.factor || 1000, 10);

            upper = Math.pow(10, ceiling);
            lower = upper / 10;

            // Make sure that max value is greater than the lower boundary
            while (max !== 0 && max < lower) {
                max *= unit;
                factor -= 1;
            }

            /*
             * And then make sure that max is lower than the upper boundary, it
             * is favored that number be less than the upper boundary than
             * higher than the lower.
             */
            while (max !== 0 && max > upper) {
                max /= unit;
                factor += 1;
            }
        }
        return factor;
    };

    /**
     * Set the auto scale information on the chart
     *
     * @access private
     * @param {number}
     *            auto scaling factor
     */
    zenoss.visualization.Chart.prototype.__configAutoScale = function(factor) {
        var scaleUnit = 1000;
        if (this.config.autoscale && this.config.autoscale.factor) {
            scaleUnit = this.config.autoscale.factor;
        }
        this.scale = {};
        this.scale.factor = factor;
        this.scale.symbol = this.__scaleSymbol(factor);
        this.scale.term = Math.pow(scaleUnit, factor);
    };

    /**
     * Formats the given value according to the format specified by the
     * configuration or a default and returns the result.
     *
     * @access private
     * @param {number}
     *            The number we are formatting
     * @param {string}
     *            The format string for example "%2f";
     */
    zenoss.visualization.Chart.prototype.formatValue = function(value) {
        var format = this.format, scaled, rval;

        /*
         * If we were given a undefined value, Infinity, of NaN (all things that
         * can't be formatted, then just return the value.
         */
        if (!$.isNumeric(value)) {
            return value;
        }
        try {
            scaled = value / this.scale.term;
            rval = sprintf(format, scaled);
            if ($.isNumeric(rval)) {
                return rval + this.scale.symbol;
            }
            // if the result is a NaN just return the original value
            return rval;
        } catch (x) {
            // override the number format for this chart
            // since this method could be called several times to render a
            // chart.
            this.format = DEFAULT_NUMBER_FORMAT;
            zenoss.visualization.__warn('Invalid format string  ' + format
                + ' using the default format.');
            scaled = value / this.scale.term;
            try {
                return sprintf(this.format, scaled) + this.scale.symbol;
            } catch (x1) {
                return scaled + this.scale.symbol;
            }
        }
    };

    /**
     * Iterates over the list of data plots and sets up display information
     * about each plot, including its legend label, color, and if it is filled
     * or not.
     *
     * @access private
     */
    zenoss.visualization.Chart.prototype.__buildPlotInfo = function() {
        var i, info, dp;

        this.plotInfo = {};
        for (i in this.config.datapoints) {
            dp = this.config.datapoints[i];
            info = {
                'legend' : dp.legend || dp.name || dp.metric,
                'color' : dp.color,
                'fill' : dp.fill
            };
            this.plotInfo[dp.name || dp.metric] = info;
        }
    };

    /**
     * Checks to see if the passed in plot is actually an overlay.
     *
     * @access private
     * @param {object}
     *            plot the object representing the plot
     * @return boolean if the plot is an overlay
     */
    zenoss.visualization.Chart.prototype.__isOverlay = function(plot) {
        var i, key = (typeof plot === 'string' ? plot : plot.key);
        if (this.overlays.length) {
            for (i = 0; i < this.overlays.length; i += 1) {
                if (this.overlays[i].legend === key) {
                    return true;
                }
            }
        }
        return false;
    };

    /**
     * Set the relative size of the chart and footer, if configured for a
     * footer, and then resizes the underlying chart.
     *
     * @access private
     */
    zenoss.visualization.Chart.prototype.__resize = function() {
        var fheight, height, span;

        fheight = this.__hasFooter() ? parseInt($(this.table).outerHeight(), 10)
            : 0;
        height = parseInt($(this.div).height(), 10) - fheight;
        span = $(this.message).find('span');

        $(this.svgwrapper).outerHeight(height);
        if (this.impl) {
            this.impl.resize(this, height);
        }

        $(this.message).outerHeight(height);
        span.css('margin-top', -parseInt(span.height(), 10) / 2);
    };

    /**
     * Constructs and appends a footer row onto the footer table
     *
     * @access private
     */
    zenoss.visualization.Chart.prototype.__appendFooterRow = function() {
        var tr, td, d, i;

        tr = document.createElement('tr');
        $(tr).addClass('zenfooter_value_row');

        // One column for the color
        td = document.createElement('td');
        $(td).addClass('zenfooter_box_column');
        d = document.createElement('div');
        $(d).addClass('zenfooter_box');
        $(d).css('backgroundColor', 'white');
        $(td).append($(d));
        $(tr).append($(td));

        // One column for the metric name
        td = document.createElement('td');
        $(td).addClass('zenfooter_data');
        $(td).addClass('zenfooter_data_text');
        $(tr).append($(td));

        // One col for each of the metrics stats
        for (i = 0; i < 4; i += 1) {
            td = document.createElement('td');
            $(td).addClass('zenfooter_data');
            $(td).addClass('zenfooter_data_number');
            $(tr).append($(td));
        }

        $(this.table).append($(tr));
        return $(tr);
    };

    zenoss.visualization.Chart.prototype.__getAssociatedPlot = function(dp) {
        var i, ll;
        if (!this.plots) {
            return undefined;
        }

        ll = this.plots.length;
        for (i = 0; i < ll; i += 1) {
            if (this.plots[i].key === (dp.legend || dp.metric)) {
                return this.plots[i];
            }
        }
        return undefined;
    };

    /**
     * Updates the chart footer based on updated data. This includes adding or
     * removing footer rows as well as filling in colors and data.
     *
     * @access private
     * @return true if the changes to the footer necesitates a resize of the
     *         chart, else false.
     */
    zenoss.visualization.Chart.prototype.__updateFooter = function(data) {
        var sta, eta, plot, dp, vals, cur, min, max, avg, cols, init, label, ll, i, v, vIdx, k, rows, row, box, color, resize = false,
            timezone = this.timezone || jstz.determine().name();
        if (!this.table) {
            return false;
        }
        rows = $(this.table).find('tr');
        if (data) {
            sta = zenoss.visualization.dateFormatter(data.startTimeActual, timezone );
            eta = zenoss.visualization.dateFormatter(data.endTimeActual, timezone);
        } else {
            sta = eta = 'N/A';

        }
        $($(rows[0]).find('td')).html(
            sta + ' to ' + eta + ' (' + timezone + ')');

        /*
         * The class on the value rows was set when they were created so get a
         * list of all those.
         */
        rows = $(this.table).find('tr.zenfooter_value_row');

        /*
         * Calculate the summary values from the data and place the date in the
         * the table.
         */
        ll = this.config.datapoints.length;
        row = 0;
        if (!this.__footerRangeOnly()) {
            for (i in this.config.datapoints) {
                dp = this.config.datapoints[i];
                plot = this.__getAssociatedPlot(dp);
                if (!this.__isOverlay(dp.legend || dp.metric)
                    && (dp.emit === undefined || dp.emit)) {
                    if (row >= rows.length) {
                        rows.push(this.__appendFooterRow());
                        resize = true;
                    }

                    // The first column is the color, the second is the metric
                    // name,
                    // followed byt the values
                    cols = $(rows[row]).find('td');

                    // footer color
                    if (this.impl) {
                        color = this.impl.color(this, this.closure, i);
                    } else {
                        color = 'white'; // unable to determine color
                    }

                    if (dp.color) {
                        color.color = dp.color;
                    }
                    box = $(cols[0]).find('div.zenfooter_box');
                    box.css('background-color', color.color);
                    box.css('opacity', color.opacity);

                    // Metric name
                    label = dp.legend || dp.metric;
                    if ((k = label.indexOf('{')) > -1) {
                        label = label.substring(0, k) + '{*}';
                    }
                    $(cols[1]).html(label);

                    if (!plot) {
                        for (v = 2; v < 6; v += 1) {
                            $(cols[v]).html('N/A');
                        }
                    } else {
                        vals = [ 0, -1, -1, 0 ];
                        cur = 0;
                        min = 1;
                        max = 2;
                        avg = 3;
                        init = false;
                        for (vIdx in plot.values) {
                            v = plot.values[vIdx];
                            if (!init) {
                                vals[min] = v.y;
                                vals[max] = v.y;
                                init = true;
                            } else {
                                vals[min] = Math.min(vals[min], v.y);
                                vals[max] = Math.max(vals[max], v.y);
                            }
                            vals[avg] += v.y;
                            vals[cur] = v.y;
                        }
                        vals[avg] = vals[avg] / plot.values.length;
                        for (v = 0; v < vals.length; v += 1) {
                            $(cols[2 + v]).html(this.formatValue(vals[v]));
                        }
                    }
                    row += 1;
                }
            }
        }

        // Extra rows exit in the table and need to be remove
        if (row < rows.length - 1) {
            for (i = rows.length - 1; i >= row; i -= 1) {
                rows[i].remove();
            }
            resize = true;
        }
        return resize;
    };

    /**
     * Returns true if this chart is displaying a footer, else false
     *
     * @access private
     * @return true if this chart is displaying a footer, else false
     */
    zenoss.visualization.Chart.prototype.__hasFooter = function() {
        return (this.config.footer === undefined
            || (typeof this.config.footer === 'boolean' && this.config.footer === true) || (typeof this.config.footer === 'string' && this.config.footer === 'range'));
    };

    /**
     * Returns true if this chart is displaying only the range in the footer,
     * else false
     *
     * @access private
     * @return true if this chart is displaying only the range in the footer,
     *         else false
     */
    zenoss.visualization.Chart.prototype.__footerRangeOnly = function() {
        return (typeof this.config.footer === 'string' && this.config.footer === 'range');
    };

    /**
     * Constructs the chart footer for a given chart. The footer will contain
     * information such as the date range and key values (ending, min, max, avg)
     * of each plot on the chart.
     *
     * @access private
     * @param {object}
     *            config the charts configuration
     * @param {object}
     *            data the data returned from the metric service that contains
     *            the data to be charted
     */
    zenoss.visualization.Chart.prototype.__buildFooter = function(config, data) {
        var tr, td, dates, th;
        this.table = document.createElement('table');
        $(this.table).addClass('zenfooter_content');
        $(this.table).addClass('zenfooter_text');
        $(this.footer).append($(this.table));

        // One row for the date range of the chart
        tr = document.createElement('tr');
        td = document.createElement('td');
        dates = document.createElement('span');
        $(td).addClass('zenfooter_dates');
        $(td).attr('colspan', 6);
        $(dates).addClass('zenfooter_dates_text');
        $(tr).append($(td));
        $(td).append($(dates));
        $(this.table).append($(tr));

        if (!this.__footerRangeOnly()) {

            // One row for the stats table header
            tr = document.createElement('tr');
            [ '', 'Metric', 'Ending', 'Minimum', 'Maximum', 'Average' ]
                .forEach(function(s) {
                    th = document.createElement('th');
                    $(th).addClass('footer_header');
                    $(th).html(s);
                    if (s.length === 0) {
                        $(th).addClass('zenfooter_box_column');
                    }
                    $(tr).append($(th));
                });
            $(this.table).append($(tr));
        }

        // Fill in the stats table
        this.__updateFooter(data);
    };

    /**
     * Updates a graph with the changes specified in the given change set. To
     * remove a value from the configuration its value should be set to a
     * negative sign, '-'.
     *
     * @param {object}
     *            changeset updates to the existing graph's configuration.
     */
    zenoss.visualization.Chart.prototype.update = function(changeset) {
        var self = this, kill = [], property;

        // This function is really meant to only handle given types of changes,
        // i.e. we don't expect that you can change the type of the graph but
        // you
        // should be able to change the date range.
        this.config = zenoss.visualization.__merge(this.config, changeset);

        // A special check for the removal of items from the config. If the
        // value
        // for any item in the change set is '-', then we delete that key.
        for (property in this.config) {
            if (this.config.hasOwnProperty(property)) {
                if (this.config[property] === '-') {
                    kill.push(property);
                }
            }
        }
        kill.forEach(function(p) {
            delete self.config[p];
        });

        /*
         * Rebuild the legend and color tables
         */
        this.__buildPlotInfo();

        try {
            this.request = this.__buildDataRequest(this.config);
            $
                .ajax({
                    'url' : zenoss.visualization.url
                        + zenoss.visualization.urlPerformance,
                    'type' : 'POST',
                    'data' : JSON.stringify(this.request),
                    'dataType' : 'json',
                    'contentType' : 'application/json',
                    'success' : function(data) {
                        var results = self.__processResult(self.request,
                            data);
                        self.plots = results[0];
                        self.__configAutoScale(results[1]);

                        /*
                         * If the chart has not been created yet, then
                         * create it, else just update the data.
                         */
                        if (!self.closure) {
                            if (self.config.type === undefined) {
                                self.config.type = 'line';
                            }
                            self.__render(data);
                        } else {
                            self.__updateData(data);
                        }

                        // Update the footer
                        if (self.__updateFooter(data)) {
                            self.__resize();
                        }
                    },
                    'error' : function(res) {
                        /*
                         * Many, many reasons that we might have gotten
                         * here, with most of them we are not able to detect
                         * why. If we have a readystate of 4 and an response
                         * code in the 200s that likely means we were unable
                         * to parse the JSON returned from the server. If
                         * not that then who knows ....
                         */
                        self.plots = undefined;
                        if (self.__updateFooter()) {
                            self.__resize();
                        }

                        var err, detail;
                        if (res.readyState === 4
                            && Math.floor(res.status / 100) === 2) {
                            detail = 'Severe: Unable to parse data returned from Zenoss metric service as JSON object. Please copy / paste the REQUEST and RESPONSE written to your browser\'s Java Console into an email to Zenoss Support';
                            zenoss.visualization
                                .__group('Severe error, please report');
                            zenoss.visualization.__error('REQUEST : POST '
                                + zenoss.visualization.urlPerformance
                                + '  ' + JSON.stringify(self.request));
                            zenoss.visualization.__error('RESPONSE: '
                                + res.responseText);
                            zenoss.visualization.__groupEnd();
                            zenoss.visualization.__showError(self.name,
                                detail);
                        } else {
                            try {
                                err = JSON.parse(res.responseText);
                                if (!err || !err.errorSoruce
                                    || !err.errorMessage) {
                                    detail = 'An unexpected failure response was received from the server. The reported message is: '
                                        + res.responseText;
                                } else {
                                    detail = 'An unexpected failure response was received from the server. The reported message is: '
                                        + err.errorSource
                                        + ' : '
                                        + err.errorMessage;
                                }
                            } catch (e) {
                                detail = 'An unexpected failure response was received from the server. The reported message is: '
                                    + res.statusText
                                    + ' : '
                                    + res.status;
                            }
                            zenoss.visualization.__error(detail);
                            zenoss.visualization.__showError(self.name,
                                detail);
                        }
                    }
                });
        } catch (x) {
            this.plots = undefined;
            if (self.__updateFooter()) {
                self.__resize();
            }
            zenoss.visualization.__error(x);
            zenoss.visualization.__showError(this.name, x);
        }
    };

    /**
     * Constructs a request object that can be POSTed to the Zenoss Data API to
     * retrieve the data for a chart. The request is based on the information in
     * the given config.
     *
     * @access private
     * @param {object}
     *            config the config from which to build a request
     * @returns {object} a request object that can be POST-ed to the Zenoss
     *          performance metric service
     */
    zenoss.visualization.Chart.prototype.__buildDataRequest = function(config) {
        var request = {};
        if (config !== undefined) {
            if (config.range !== undefined) {
                if (config.range.start !== undefined) {
                    request.start = config.range.start;
                }
                if (config.range.end !== undefined) {
                    request.end = config.range.end;
                }
            }

            if (config.series !== undefined) {
                request.series = config.series;
            }

            if (config.downsample !== undefined) {
                request.downsample = config.downsample;
            }

            if (config.tags !== undefined) {
                request.tags = config.tags;
            }

            if (config.returnset !== undefined) {
                request.returnset = config.returnset;
            }

            if (config.grouping !== undefined) {
                request.grouping = config.grouping;
            }

            if (config.datapoints !== undefined) {
                request.metrics = [];
                config.datapoints
                    .forEach(function(dp) {
                        var m = {}, key;
                        if (dp.metric !== undefined) {
                            m.metric = dp.metric;

                            if (dp.rate !== undefined) {
                                m.rate = dp.rate;
                            }
                            if (dp.aggregator !== undefined) {
                                m.aggregator = dp.aggregator;
                            }

                            if (dp.tags !== undefined) {
                                m.tags = {};
                                for (key in dp.tags) {
                                    if (dp.tags.hasOwnProperty(key)) {
                                        m.tags[key] = dp.tags[key];
                                    }
                                }
                            }

                            if (dp.name === undefined) {
                                m.name = dp.metric;
                            } else {
                                m.name = dp.name;
                            }
                        } else if (dp.name !== undefined) {
                            m.name = dp.name;
                        } else {
                            /*
                             * This data point has neither a metric
                             * definition nor a name (virtual metric)
                             * deffined. As such this is an invalid
                             * specification. Because of this we will fail
                             * the entire request so that the caller is not
                             * confused as to why partial data is returned.
                             */
                            throw sprintf(
                                "Invalid data point specification in request, '%s'. No 'metric' or 'name' attribute specified, failing entire request.",
                                JSON.stringify(dp, null, ' '));
                        }

                        if (dp.downsample !== undefined) {
                            m.downsample = dp.downsample;
                        }
                        if (dp.expression !== undefined) {
                            m.expression = dp.expression;
                        }
                        if (dp.emit !== undefined) {
                            m.emit = dp.emit;
                        }
                        request.metrics.push(m);
                    });

            }
        }
        return request;
    };

    /**
     * Processes the result from the Zenoss performance metric query that is in
     * the series format into the data that can be utilized by the chart
     * library.
     *
     * @access private
     * @param {object}
     *            request the request which generated the data
     * @param {object}
     *            data the data object returned from the query
     * @returns {object} the data in the format that can be utilized by the
     *          chart library.
     */
    zenoss.visualization.Chart.prototype.__processResultAsSeries = function(
        request, data) {
        var self = this, plots = [], max = 0, i, result, dpi, dp, info, key, plot, tag, prefix;

        for (i in data.results) {
            result = data.results[i];

            /*
             * The key for a series plot will be its distinguishing
             * characteristics, which is the metric name and the tags. We will
             * use any mapping from metric name to legend value that was part of
             * the original request.
             */
            info = self.plotInfo[result.metric];
            key = info.legend;
            if (result.tags !== undefined) {
                key += '{';
                prefix = '';
                for (tag in result.tags) {
                    if (result.tags.hasOwnProperty(tag)) {
                        key += prefix + tag + '=' + result.tags[tag];
                        prefix = ',';
                    }
                }
                key += '};';
            }
            plot = {
                'key' : key,
                'color' : info.color,
                'fill' : info.fill,
                'values' : []
            };
            for (dpi in result.datapoints) {
                dp = result.datapoints[dpi];
                max = Math.max(Math.abs(dp.value), max);
                plot.values.push({
                    'x' : dp.timestamp * 1000,
                    'y' : dp.value
                });
            }
            plots.push(plot);
        }

        return [ plots, this.__calculateAutoScaleFactor(max) ];
    };

    /**
     * Processes the result from the Zenoss performance metric query that is in
     * the default format into the data that can be utilized by the chart
     * library.
     *
     * @access private
     * @param {object}
     *            request the request which generated the data
     * @param {object}
     *            data the data object returned from the query
     * @returns {object} the data in the format that can be utilized by the
     *          chart library.
     */
    zenoss.visualization.Chart.prototype.__processResultAsDefault = function(
        request, data) {

        var self = this, plotMap = {}, i, result, max = 0, info, plot, plots, key, xcompare;

        /*
         * Create a plot for each metric name, this is essentially grouping the
         * results by metric name. This can cause problems if the request
         * contains multiple queries for the same metric, but this is basically
         * a restriction of the implementation (OpenTSDB) where it doesn't split
         * the results logically when multiple requests are made in a single
         * call.
         */
        for (i in data.results) {
            result = data.results[i];
            plot = plotMap[result.metric];
            if (plot === undefined) {
                info = self.plotInfo[result.metric];
                plot = {
                    'key' : info.legend,
                    'color' : info.color,
                    'fill' : info.fill,
                    'values' : []
                };
                plotMap[result.metric] = plot;
            }

            max = Math.max(Math.abs(result.value), max);
            plot.values.push({
                'x' : result.timestamp * 1000,
                'y' : result.value
            });
        }

        xcompare = function(a, b) {
            if (a.x < b.x) {
                return -1;
            }
            if (a.x > b.x) {
                return 1;
            }
            return 0;
        };

        /*
         * Convert the plotMap into an array of plots for the graph library to
         * process
         */
        plots = [];
        for (key in plotMap) {
            if (plotMap.hasOwnProperty(key)) {
                // Sort the values of the plot as we put them in the
                // plots aray.
                plotMap[key].values.sort(xcompare);
                plots.push(plotMap[key]);
            }
        }
        return [ plots, this.__calculateAutoScaleFactor(max) ];
    };

    /**
     * Wrapper function that redirects to the proper implementation to processes
     * the result from the Zenoss performance metric query into the data that
     * can be utilized by the chart library. *
     *
     * @access private
     * @param {object}
     *            request the request which generated the data
     * @param {object}
     *            data the data object returned from the query
     * @returns {object} the data in the format that can be utilized by the
     *          chart library.
     */
    zenoss.visualization.Chart.prototype.__processResult = function(request,
                                                                    data) {
        var results, plots, i, overlay, minDate, maxDate, plot, k, firstMetric;

        if (data.series) {
            results = this.__processResultAsSeries(request, data);
            plots = results[0];
        } else {
            results = this.__processResultAsDefault(request, data);
            plots = results[0];
        }

        // add overlays
        if (this.overlays.length && plots.length) {
            for (i in this.overlays) {
                overlay = this.overlays[i];
                // get the date range
                firstMetric = plots[0];
                plot = {
                    'key' : overlay.legend,
                    'disabled' : true,
                    'values' : [],
                    'color' : overlay.color
                };
                minDate = firstMetric.values[0].x;
                maxDate = firstMetric.values[firstMetric.values.length - 1].x;
                for (k = 0; k < overlay.values.length; k += 1) {

                    // create a line by putting a point at the start and a point
                    // at the end
                    plot.values.push({
                        x : minDate,
                        y : overlay.values[k]
                    });
                    plot.values.push({
                        x : maxDate,
                        y : overlay.values[k]
                    });
                }
                plots.push(plot);
            }
        }
        return results;
    };

    /**
     * Deep object merge. This merge differs significantly from the "extend"
     * method provide by jQuery in that it will merge the value of arrays, but
     * concatenating the arrays together using the jQuery method "merge".
     * Neither of the objects passed are modified and a new object is returned.
     *
     * @access private
     * @param {object}
     *            base the object to which values are to be merged into
     * @param {object}
     *            extend the object from which values are merged
     * @returns {object} the merged object
     */
    zenoss.visualization.__merge = function(base, extend) {
        var m, k, v;
        if (zenoss.visualization.debug) {
            zenoss.visualization.__groupCollapsed('Object Merge');
            zenoss.visualization.__group('SOURCES');
            zenoss.visualization.__log(base);
            zenoss.visualization.__log(extend);
            zenoss.visualization.__groupEnd();
        }

        if (base === undefined || base === null) {
            m = $.extend(true, {}, extend);
            if (zenoss.visualization.debug) {
                zenoss.visualization.__log(m);
                zenoss.visualization.__groupEnd();
            }
            return m;
        }
        if (extend === undefined || extend === null) {
            m = $.extend(true, {}, base);
            if (zenoss.visualization.debug) {
                zenoss.visualization.__log(m);
                zenoss.visualization.__groupEnd();
            }
            return m;
        }

        m = $.extend(true, {}, base);
        for (k in extend) {
            if (extend.hasOwnProperty(k)) {
                v = extend[k];
                if (v.constructor === Number || v.constructor === String) {
                    m[k] = v;
                } else if (v instanceof Array) {
                    m[k] = $.merge(m[k], v);
                } else if (v instanceof Object) {
                    if (m[k] === undefined) {
                        m[k] = $.extend({}, v);
                    } else {
                        m[k] = zenoss.visualization.__merge(m[k], v);
                    }
                } else {
                    m[k] = $.extend(m[k], v);
                }
            }
        }

        if (zenoss.visualization.debug) {
            zenoss.visualization.__log(m);
            zenoss.visualization.__groupEnd();
        }
        return m;
    };

    /**
     * Given a dependency object, checks if the dependencies are already loaded
     * and if so, calls the callback, else loads the dependencies and then calls
     * the callback.
     *
     * @access private
     * @param {object}
     *            required the dependency object that contains a "defined" key
     *            and a "source" key. The "defined" key is a name (string) that
     *            is used to identify the dependency and the "source" key is an
     *            array of JavaScript and CSS URIs that must be loaded to meet
     *            the dependency.
     * @param {function}
     *            callback called after the dependencies are loaded
     */
    zenoss.visualization.__loadDependencies = function(required, callback) {
        var base, o, c, js, css;

        if (required === undefined) {
            callback();
            return;
        }

        // Check if it is already loaded, using the value in the 'defined' field
        if (zenoss.visualization.__dependencies[required.defined] !== undefined
            && zenoss.visualization.__dependencies[required.defined].state !== undefined) {
            o = zenoss.visualization.__dependencies[required.defined].state;
        }
        if (o !== undefined) {
            if (o === 'loaded') {
                if (zenoss.visualization.debug) {
                    zenoss.visualization.__log('Dependencies for "'
                        + required.defined
                        + '" already loaded, continuing.');
                }
                // Already loaded, so just invoke the callback
                callback();
            } else {
                // It is in the process of being loaded, so add our callback to
                // the
                // list of callbacks to call when it is loaded.
                if (zenoss.visualization.debug) {
                    zenoss.visualization
                        .__log('Dependencies for "'
                            + required.defined
                            + '" in process of being loaded, queuing until loaded.');
                }

                c = zenoss.visualization.__dependencies[required.defined].callbacks;
                c.push(callback);
            }
            return;
        } else {
            // OK, not yet loaded or being loaded, so it is ours.
            if (zenoss.visualization.debug) {
                zenoss.visualization
                    .__log('Dependencies for "'
                        + required.defined
                        + '" not loaded nor in process of loading, initiate loading.');
            }

            zenoss.visualization.__dependencies[required.defined] = {};
            base = zenoss.visualization.__dependencies[required.defined];
            base.state = 'loading';
            base.callbacks = [];
            base.callbacks.push(callback);
        }

        // Load the JS and CSS files. Divide the list of files into two lists:
        // JS
        // and CSS as we can load one async, and the other loads sync (CSS).
        js = [];
        css = [];
        required.source.forEach(function(v) {
            if (v.endsWith('.js')) {
                js.push(v);
            } else if (v.endsWith('.css')) {
                css.push(v);
            } else {
                zenoss.visualization.__warn('Unknown required file type, "' + v
                    + '" when loading dependencies for "' + 'unknown'
                    + '". Ignored.');
            }
        });

        base = zenoss.visualization.__dependencies[required.defined];
        zenoss.visualization.__load(js, css, function() {
            base.state = 'loaded';
            base.callbacks.forEach(function(c) {
                c();
            });
            base.callbacks.length = 0;
        });
    };

    /**
     * Returns true if the chart has plots and they contain data points, else
     * false.
     *
     * @access private
     */
    zenoss.visualization.Chart.prototype.__havePlotData = function() {
        var i, ll;

        if (!this.plots || this.plots.length === 0) {
            return false;
        }

        ll = this.plots.length;
        for (i = 0; i < ll; i += 1) {
            if (this.plots[i].values.length > 0) {
                return true;
            }
        }
        return false;
    };

    /**
     * Updates the chart with a new data set
     *
     * @access private
     * @param {object}
     *            the new data to display in the chart
     */
    zenoss.visualization.Chart.prototype.__updateData = function(data) {
        if (!this.__havePlotData()) {
            zenoss.visualization.__showNoData(this.name);
        } else {
            zenoss.visualization.__showChart(this.name);
            this.impl.update(this, data);
        }
        if (this.__updateFooter(data)) {
            this.__resize();
        }
    };

    /**
     * Constructs a chart from the given data
     *
     * @param data
     *            the data returned from a metric query
     * @access private
     */
    zenoss.visualization.Chart.prototype.__buildChart = function(data) {
        $(this.svgwrapper).outerHeight(
            $(this.div).height() - $(this.footer).outerHeight());
        this.closure = this.impl.build(this, data);
        this.impl.render(this);

        // If there is not data, let the user know
        if (!this.__havePlotData()) {
            zenoss.visualization.__showNoData(this.name);
        }
    };

    /**
     * Loads the chart renderer as a dependency and then constructs and renders
     * the chart.
     *
     * @access private
     * @param {object}
     *            data the data that is being rendered in the graph
     */
    zenoss.visualization.Chart.prototype.__render = function(data) {
        var self = this;
        zenoss.visualization
            .__loadDependencies({
                'defined' : self.config.type.replace('.', '_'),
                'source' : [ 'charts/' + self.config.type.replace('.', '/')
                    + '.js' ]
            },
            function() {
                var i;
                try {
                    i = zenoss.visualization.chart;
                    self.config.type.split('.').forEach(
                        function(seg) {
                            i = i[seg];
                        });
                    self.impl = i;
                } catch (err) {
                    throw new zenoss.visualization.Error(
                        'DependencyError',
                        'Unable to locate loaded chart type, "'
                            + self.config.type
                            + '", error: ' + err);
                }

                // Check the impl to see if a dependency is listed
                // and
                // if so load that.
                zenoss.visualization.__loadDependencies(
                    self.impl.required, function() {
                        self.__buildChart(data);
                        if (self.__hasFooter()) {
                            self.__buildFooter(self.config,
                                data);
                        }
                        self.__resize();
                    });
            });
    };

    /**
     * Loads the CSS specified by the URL.
     *
     * @access private
     * @param {url}
     *            url the url, in string format, of the CSS file to load.
     */
    zenoss.visualization.__loadCSS = function(url) {
        var css = document.createElement('link');
        css.rel = 'stylesheet';
        css.type = 'text/css';

        if (!url.startsWith("http")) {
            css.href = zenoss.visualization.url + zenoss.visualization.urlPath
                + url;
        } else {
            css.href = url;
        }
        document.getElementsByTagName('head')[0].appendChild(css);
    };

    /**
     * We would like to use jQuery for dynamic loading of JavaScript files, but
     * it may be that jQuery is not yet loaded, so we first have to dynamically
     * load jQuery. To accomplish this we need a 'bootstrap' loader. This method
     * will load the JavaScript file specified by the URL by creating a new HTML
     * script element on the page and then call the callback once the script has
     * been loaded.
     *
     * @access private
     * @param {url}
     *            url URL, in string form, of the JavaScript file to load
     * @param {function}
     *            callback the function to call once the JavaScript is loaded
     */
    zenoss.visualization.__bootstrapScriptLoader = function(url, callback) {
        var script, deferred, _callback;

        function ZenDeferred() {
            var failCallback;

            this.fail = function(_) {
                if (!arguments.length) {
                    return failCallback;
                }
                failCallback = _;
                return failCallback;
            };
        }

        script = document.createElement("script");
        deferred = new ZenDeferred();
        _callback = callback;
        script.type = "text/javascript";
        script.async = true;

        if (script.readyState) { // IE
            script.onreadystatechange = function() {
                if (script.readyState === "loaded") {
                    var fail = deferred.fail();
                    if (fail !== undefined && fail !== null) {
                        fail();
                    }
                }
                if (script.readyState === "complete") {
                    script.onreadystatechange = null;
                    _callback();
                }
            };
        } else { // Others
            script.onload = function() {
                _callback();
            };
            script.onerror = function(e) {
                var fail = deferred.fail();
                if (fail !== undefined && fail !== null) {
                    fail(undefined, undefined, e.type);
                }
            };
        }

        script.src = url;
        document.getElementsByTagName("head")[0].appendChild(script);
        return deferred;
    };

    /**
     * Loads the array of JavaScript URLs followed by the array of CSS URLs and
     * calls the appropriate callback if the operations succeeded or failed.
     *
     * @access private
     * @param {uri[]}
     *            js an array of JavaScript files to load
     * @param {uri[]}
     *            css an array of CSS files to load
     * @param {function}
     *            [success] callback to call one everything is loaded
     * @param {function}
     *            [fail] callback to call if there is a failure
     */
    zenoss.visualization.__load = function(js, css, success, fail) {
        if (zenoss.visualization.debug) {
            zenoss.visualization.__log('Request to load "' + js + '" and "'
                + css + '".');
        }
        if (js.length === 0) {
            // All JavaScript files are loaded, now the loading of CSS can begin
            css.forEach(function(uri) {
                zenoss.visualization.__loadCSS(uri);
                if (zenoss.visualization.debug) {
                    zenoss.visualization.__log('Loaded dependency "' + uri
                        + '".');
                }
            });
            if (typeof success === 'function') {
                success();
            }
            return;
        }

        // Shift the next value off of the JS array and load it.
        var uri = js.shift(), _js = js, _css = css, self = this, loader;
        if (!uri.startsWith("http")) {
            uri = zenoss.visualization.url + zenoss.visualization.urlPath + uri;
        }

        loader = zenoss.visualization.__bootstrapScriptLoader;
        if (window.jQuery !== undefined) {
            loader = $.getScript;
        }
        loader(uri, function() {
            if (zenoss.visualization.debug) {
                zenoss.visualization.__log('Loaded dependency "' + uri + '".');
            }
            self.__load(_js, _css, success, fail);
        })
            .fail(
            function(_1, _2, exception) {
                zenoss.visualization
                    .__error('Unable to load dependency "'
                        + uri
                        + '", with error "'
                        + exception
                        + '". Loading halted, will continue, but additional errors likely.');
                self.__load(_js, _css, success, fail);
            });
    };

    /**
     * Loads jQuery and D3 as a dependencies and then calls the appripriate
     * callback.
     *
     * @access private
     * @param {function}
     *            [success] called if the core dependencies are loaded
     * @param {function}
     *            [fail] called if the core dependencies are not loaded
     */
    zenoss.visualization.__bootstrap = function(success, fail) {
        var sources = [ 'jquery.min.js', 'd3.v3.min.js', 'jstz-1.0.4.min.js',
            'css/zenoss.css', 'sprintf.min.js' ];
        // if moment isn't already loaded load it now, hopefully the versions will be compatible
        if (window.moment == null) {
            sources = sources.concat(['moment.min.js', 'moment-timezone.js', 'moment-timezone-data.js']);
        }
        zenoss.visualization.__loadDependencies({
            'defined' : 'd3',
            'source' : sources
        }, success, fail);
    };
    window.zenoss = zenoss;
}(window));
