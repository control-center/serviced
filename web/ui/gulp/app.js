/* jshint node: true */

var gulp = require("gulp"),
    sequence = require("run-sequence"),
    babel = require("gulp-babel"),
    sourcemaps = require("gulp-sourcemaps"),
    uglify = require("gulp-uglify"),
    rename = require("gulp-rename"),
    jshint = require("gulp-jshint"),
    concat = require("gulp-concat"),
    merge = require("merge-stream");

var config = require("./config.js");

gulp.task("build", () => {
    sequence("lint", "concat", "copyStatic", () => {});
});

gulp.task("release", function(){
    // last arg is a callback function in case
    // of an error.
    sequence("lint", "concat", "uglify", "copyStatic", function(){});
});

gulp.task("concat", function(){
    return gulp.src(config.controlplaneFiles)
        .pipe(sourcemaps.init())
            .pipe(babel(config.babelConfig))
            .pipe(concat("controlplane.js"))
        .pipe(sourcemaps.write("./", { sourceRoot: "src" }))
        .pipe(gulp.dest(config.paths.srcBuild));
});

gulp.task("uglify", function(){
    return gulp.src(config.paths.build + "controlplane.js")
        .pipe(sourcemaps.init({loadMaps: true}))
            .pipe(uglify())
        .pipe(sourcemaps.write("./"))
        .pipe(gulp.dest(config.paths.srcBuild));
});

gulp.task("copyStatic", function() {
    let a = gulp.src(config.staticFiles, {base: config.paths.staticSrc})
        .pipe(gulp.dest(config.paths.staticBuild));

    // gather partials from src
    let b = gulp.src(config.controlplanePartials)
        .pipe(rename({dirname: ""}))
        .pipe(gulp.dest(config.paths.staticBuild + "partials"));

    return merge(a,b);
});

gulp.task("lint", function(){
    return gulp.src(config.controlplaneFiles)
        .pipe(jshint(".jshintrc"))
        .pipe(jshint.reporter("jshint-stylish"))
        .pipe(jshint.reporter("fail"));
});

