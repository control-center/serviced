/* jshint node: true */
var gulp = require("gulp"),
    sequence = require("run-sequence"),
    sourcemaps = require("gulp-sourcemaps"),
    uglify = require("gulp-uglify"),
    concat = require("gulp-concat");

var config = require("./config.js");

// this needs to be run manually if 3rd party
// code is updated, which should be infrequent
gulp.task("release3rdparty", cb => {
    sequence("copyStatic", "concat3rdparty", "uglify3rdparty", cb);
});

gulp.task("debug3rdparty", cb => {
    sequence("copyStatic", "concat3rdparty", "copyStatic", cb);
});

gulp.task("concat3rdparty", function(){
    return gulp.src(config.thirdpartyFiles)
        .pipe(sourcemaps.init())
            .pipe(concat("thirdparty.js"))
        .pipe(sourcemaps.write("./", { sourceRoot: "thirdParty" }))
        .pipe(gulp.dest(config.paths.thirdpartyBuild));
});

gulp.task("uglify3rdparty", function(){
    return gulp.src(config.paths.thirdpartyBuild + "thirdparty.js")
        .pipe(sourcemaps.init({loadMaps: true}))
            .pipe(uglify())
        .pipe(sourcemaps.write("./"))
        .pipe(gulp.dest(config.paths.thirdpartyBuild));
});

