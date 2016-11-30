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
 * here are the common tasks
 *
 * default
 * builds the js library.
 *
 * watch
 * watches the filesystem and continuously builds the js lib
 * NOTE: watch does NOT transpile the js, so you will need
 * a modern browser for things to work
 *
 * test
 * runs unit tests
 *
 * tdd
 * watches the filesystem and continuously builds and runs
 * the unit tests
 */
