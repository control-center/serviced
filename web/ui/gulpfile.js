/* jshint node: true */

let gulp = require("gulp"),
    clean = require("gulp-clean"),
    paths = require("./gulp/config.js").paths;

// get all the gulp tasks
require("./gulp/app.js");
require("./gulp/thirdparty.js");
require("./gulp/test.js");
require("./gulp/watch.js");

gulp.task("default", ["build"]);

gulp.task("clean", () => {
    return gulp.src(paths.build + "*", {read: false})
        .pipe(clean());
});

/*
 * you probably want to do one of these:
 *
 * `gulp`
 * lints and builds the js library. NOTE: if you're using
 * a browser that supports most es6, you can speed this up
 * by setting SKIP_TRANSPILE. eg: `SKIP_TRANSPILE=1 gulp`
 *
 * `gulp watch`
 * watches the filesystem and continuously builds the js lib
 * NOTE: if you're using a browser that supports most es6,
 * you can speed this up by setting SKIP_TRANSPILE.
 * eg: `SKIP_TRANSPILE=1 gulp watch`
 *
 * `gulp test`
 * runs unit tests
 *
 * `gulp tdd`
 * watches the filesystem and continuously builds and runs
 * the unit tests
 */
