/* jshint node: true */

let gulp = require("gulp"),
    config = require("./gulp/config.js"),
    paths = config.paths;

gulp.task("watch", function(){
    // concat js
    gulp.watch(paths.src + "/**/*.js", ["concat"]);
    // copy html templates
    gulp.watch(paths.src + "/**/*.html", ["copyStatic"]);
    // copy static content
    gulp.watch(config.staticFiles, ["copyStatic"]);
    // copy translations
    gulp.watch(paths.staticSrc + "/i18n/*", ["copyStatic"]);
    // TODO - preprocess CSS
});

