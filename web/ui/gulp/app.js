/* jshint node: true */

var gulp = require("gulp"),
    sequence = require("run-sequence"),
    babel = require("gulp-babel"),
    sourcemaps = require("gulp-sourcemaps"),
    uglify = require("gulp-uglify"),
    rename = require("gulp-rename"),
    jshint = require("gulp-jshint"),
    concat = require("gulp-concat"),
    gulpif = require("gulp-if"),
    merge = require("merge-stream");

var config = require("./config.js");

var babelConfig = {
    presets: ["es2015"],
};

// if SKIP_TRANSPILE is set, skip js transpilation
// NOTE: browser must support es6 stuff
let shouldTranspile = !process.env.SKIP_TRANSPILE;

gulp.task("build", cb => {
    sequence("babel", "copyStatic", cb);
});

gulp.task("release", cb => {
    sequence("lint", "babel", "uglify", "copyStatic", cb);
});

gulp.task("babel", function(){
    return gulp.src(config.controlplaneFiles)
        .pipe(sourcemaps.init())
            .pipe(gulpif(shouldTranspile, babel(babelConfig)))
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

