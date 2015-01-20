/* globals require: true */

var gulp = require("gulp"),
    concat = require("gulp-concat"),
    uglify = require("gulp-uglify"),
    rename = require("gulp-rename"),
    jshint = require("gulp-jshint"),
    sequence = require("run-sequence");

var paths = {
    // TODO - organize by feature, not type
    src: "static/js/",
    controllers: "static/js/controllers/",
    modules: "static/js/modules/",
    build: "static/js/"
};

// files to be concatted/minified to make
// controlplane.js
var controlplaneFiles = [
    paths.src + "main.js",
    paths.modules + "*.js",
    paths.controllers + "*.js"
];

gulp.task("default", ["concat"]);
gulp.task("release", function(){
    // last arg is a callback function in case
    // of an error.
    sequence("lint", "concat", "uglify", function(){});
});

gulp.task("concat", function(){
    return gulp.src(controlplaneFiles)
        .pipe(concat("controlplane.js"))
        .pipe(gulp.dest(paths.build));
});

gulp.task("uglify", function(){
    return gulp.src(paths.build + "controlplane.js")
        .pipe(uglify())
        .pipe(rename(paths.build + "controlplane.min.js"))
        .pipe(gulp.dest("./"));
});

gulp.task("watch", function(){
    gulp.watch(paths.controllers + "/*", ["concat"]);
    gulp.watch(paths.modules + "/*", ["concat"]);
    gulp.watch(paths.src + "/main.js", ["concat"]);
});

gulp.task("test", function(){
    // TODO - unit tests
    // TODO - functional tests
});

gulp.task("lint", function(){
    return gulp.src(controlplaneFiles)
        .pipe(jshint(".jshintrc"))
        .pipe(jshint.reporter("jshint-stylish"))
        .pipe(jshint.reporter("fail"));
});

