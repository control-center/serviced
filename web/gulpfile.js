var gulp = require("gulp"),
    concat = require("gulp-concat"),
    uglify = require("gulp-uglify"),
    rename = require("gulp-rename"),
    jshint = require("gulp-jshint");

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

    // modules
    // TODO - organize by feature, not type!
    paths.modules + "graphPanelDirective.js",
    paths.modules + "healthIconDirective.js",
    paths.modules + "modalService.js",
    paths.modules + "resourcesFactory.js",
    paths.modules + "serviceHealth.js",
    paths.modules + "servicesFactory.js",
    paths.modules + "uiDateTimePicker.js",
    paths.modules + "zenNotify.js",
    paths.modules + "authService.js",
    paths.modules + "miscDirectives.js",

    // general controllers
    // TODO - Language shouldn't be a controller... it
    // should probably be a provider
    paths.controllers + "LanguageController.js",
    paths.controllers + "NavbarController.js",

    // specific route controllers
    paths.controllers + "BackupRestoreController.js",

    paths.controllers + "HostsController.js",
    paths.controllers + "HostDetailsController.js",
    paths.controllers + "HostsMapController.js",

    paths.controllers + "LogController.js",

    paths.controllers + "LoginController.js",

    paths.controllers + "PoolsController.js",
    paths.controllers + "PoolDetailController.js",

    paths.controllers + "AppsController.js",
    paths.controllers + "DeployWizard.js",
    paths.controllers + "ServicesMapController.js",
    paths.controllers + "ServiceDetailsController.js"
];

gulp.task("default", ["concat"]);
gulp.task("release", ["lint", "concat", "uglify"]);

gulp.task("concat", function(){
    gulp.src(controlplaneFiles)
        .pipe(concat("controlplane.js"))
        .pipe(gulp.dest(paths.build));

});

gulp.task("uglify", function(){
    // TODO - ensure controlplane.js exists
    gulp.src(paths.build + "controlplane.js")
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
    gulp.src(controlplaneFiles)
        .pipe(jshint())
        .pipe(jshint.reporter("jshint-stylish"))
        .pipe(jshint.reporter("fail"));
});

