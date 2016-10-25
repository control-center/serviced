/* jshint node: true */

let gulp = require("gulp");

// get all the gulp tasks
require("./gulp/app.js");
require("./gulp/thirdparty.js");
require("./gulp/test.js");
require("./gulp/watch.js");

gulp.task("default", ["build"]);

/*
 * you probably want to do one of these:
 *
 * `gulp`
 * lints and builds the js library
 *
 * `gulp watch`
 * watches the filesystem and continuously builds the js lib
 *
 * `gulp test`
 * runs unit tests
 *
 * `gulp tdd`
 * watches the filesystem and continuously builds and runs
 * the unit tests
 */
