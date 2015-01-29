/* globals require: true */

var gulp = require("gulp"),
    gutil = require('gulp-util'),
    concat = require("gulp-concat"),
    uglify = require("gulp-uglify"),
    jshint = require("gulp-jshint"),
    sequence = require("run-sequence"),
    to5 = require("gulp-6to5"),
    sourcemaps = require("gulp-sourcemaps"),
    karma = require('karma').server;

var paths = {
    // TODO - organize by feature, not type
    src: "src/",
    controllers: "src/controllers/",
    modules: "src/modules/",
    build: "build/js/",
    thirdpartySrc: "static/thirdparty/",
    thirdpartyBuild: "static/thirdparty/"
};

var to5Config = {
    format: {
        parentheses: true,
        comments: true,
        compact: false,
        indent: {
            adjustMultilineComment: false,
            style: "    ",
            base: 0
        }
    }
};

// files to be concatenated/minified to make
// controlplane.js
var controlplaneFiles = [
    paths.src + "main.js",
    paths.modules + "*.js",
    paths.controllers + "*.js"
];

// Third-party library files to be concatenated/minified to make thirdparty.js
var thirdpartyFiles = [
    paths.thirdpartySrc + "jquery/jquery.js",
    paths.thirdpartySrc + "jquery-timeago/jquery.timeago.js",
    paths.thirdpartySrc + "jquery-ui/ui/jquery-ui.js",
    paths.thirdpartySrc + "jquery-datetimepicker/jquery.datetimepicker.js",

    paths.thirdpartySrc + "bootstrap/dist/js/bootstrap.js",
    paths.thirdpartySrc + "bootstrap/js/tooltip.js",
    paths.thirdpartySrc + "bootstrap/js/popover.js",

    paths.thirdpartySrc + "elastic/elasticsearch.js",

    paths.thirdpartySrc + "angular/angular.js",
    paths.thirdpartySrc + "angular/angular-route.js",
    paths.thirdpartySrc + "angular/angular-cookies.js",
    paths.thirdpartySrc + "angular-dragdrop/angular-dragdrop.js",
    paths.thirdpartySrc + "angular-translate/angular-translate.js",
    paths.thirdpartySrc + "angular-translate/service/loader-static-files.js",
    paths.thirdpartySrc + "angular-translate/service/loader-url.js",
    paths.thirdpartySrc + "angular-cache/angular-cache.js",
    paths.thirdpartySrc + "angular-moment/angular-moment.js",
    paths.thirdpartySrc + "angular-sticky/sticky.js",

    paths.thirdpartySrc + "d3/d3.js",
    paths.thirdpartySrc + "graphlib/graphlib.js",
    paths.thirdpartySrc + "dagre-d3/dagre-d3.js",

    paths.thirdpartySrc + "codemirror/lib/codemirror.js",
    paths.thirdpartySrc + "codemirror/mode/properties/properties.js",
    paths.thirdpartySrc + "codemirror/mode/yaml/yaml.js",
    paths.thirdpartySrc + "codemirror/mode/xml/xml.js",
    paths.thirdpartySrc + "codemirror/mode/shell/shell.js",
    paths.thirdpartySrc + "codemirror/mode/javascript/javascript.js",
    paths.thirdpartySrc + "angular-ui-codemirror/ui-codemirror.js",
];

gulp.task("default", ["concat"]);
gulp.task("release", function(){
    // last arg is a callback function in case
    // of an error.
    sequence("lint", "concat", "uglify", function(){});
});

// this needs to run 3rd party code is
// updated, which should be infrequent
gulp.task("release3rdparty", function(){
    sequence("concat3rdparty", "uglify3rdparty", function(){});
});

gulp.task("concat", function(){
    return gulp.src(controlplaneFiles)
        .pipe(sourcemaps.init())
            .pipe(to5(to5Config))
            .pipe(concat("controlplane.js"))
        .pipe(sourcemaps.write("./", { sourceRoot: "src" }))
        .pipe(gulp.dest(paths.build));
});

gulp.task("uglify", function(){
    return gulp.src(paths.build + "controlplane.js")
        .pipe(sourcemaps.init({loadMaps: true}))
            .pipe(uglify())
        .pipe(sourcemaps.write("./"))
        .pipe(gulp.dest(paths.build));
});

gulp.task("concat3rdparty", function(){
    return gulp.src(thirdpartyFiles)
        .pipe(sourcemaps.init())
            .pipe(concat("thirdparty.js"))
        .pipe(sourcemaps.write("./", { sourceRoot: "thirdParty" }))
        .pipe(gulp.dest(paths.thirdpartyBuild));
});

gulp.task("uglify3rdparty", function(){
    return gulp.src(paths.thirdpartyBuild + "thirdparty.js")
        .pipe(sourcemaps.init({loadMaps: true}))
            .pipe(uglify())
        .pipe(sourcemaps.write("./"))
        .pipe(gulp.dest(paths.thirdpartyBuild));
});

gulp.task("watch", function(){
    gulp.watch(paths.controllers + "/*", ["concat"]);
    gulp.watch(paths.modules + "/*", ["concat"]);
    gulp.watch(paths.src + "/main.js", ["concat"]);
});

//
// The equivalent manual execution of karma is:
//  ./node_modules/karma/bin/karma start karma.conf.js --single-run \
//      --log-level debug \\
//      --browsers PhantomJS \\
//      --reporters progress,junit,coverage
gulp.task('test', function (done) {
  karma.start({
    configFile: __dirname + '/karma.conf.js',
    singleRun: true,
    logLevel: "debug",
    browsers: ["PhantomJS"],
    reporters: ["progress","junit","coverage","threshold"],
  }, function(exitStatus) {
    // Workaround for 'formatError' based on suggestions from
    //   http://stackoverflow.com/questions/26614738/issue-running-karma-task-from-gulp
    // but tweaked to use (apparently new) PluginError
    var err = exitStatus ? new gutil.PluginError('test', 'There are failing unit tests') : undefined;
    done(err);
  });
});


gulp.task("lint", function(){
    return gulp.src(controlplaneFiles)
        .pipe(jshint(".jshintrc"))
        .pipe(jshint.reporter("jshint-stylish"))
        .pipe(jshint.reporter("fail"));
});

