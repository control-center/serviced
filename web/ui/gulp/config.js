/* jshint node: true */

var paths = {
    src: "src/",
    build: "build/",
    srcBuild: "build/js/",
    staticSrc: "static/",
    staticBuild: "build/",
    thirdpartySrc: "static/thirdparty/",
    thirdpartyBuild: "static/thirdparty/"
};

// files to be concatenated/minified to make
// controlplane.js
var controlplaneFiles = [
    paths.src + "**/*.js"
];

var controlplanePartials = [
    paths.src + "**/*.html"
];

// Third-party library files to be concatenated/minified to make thirdparty.js
// NOTE: any changes here will not take effect until `release3rdparty` or `concat3rdparty`
// task is run. Be sure to commit the new *MINIFIED* thirdparty.js file.
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
    paths.thirdpartySrc + "angular/angular-sanitize.js",
    paths.thirdpartySrc + "angular-dragdrop/angular-dragdrop.js",
    paths.thirdpartySrc + "angular-translate/angular-translate.js",
    paths.thirdpartySrc + "angular-translate/angular-translate-loader-static-files/angular-translate-loader-static-files.js",
    paths.thirdpartySrc + "angular-translate/angular-translate-loader-url/angular-translate-loader-url.js",
    paths.thirdpartySrc + "angular-cache/angular-cache.js",
    paths.thirdpartySrc + "angular-moment/angular-moment.js",
    paths.thirdpartySrc + "angular-sticky/sticky.js",
    paths.thirdpartySrc + "angular-location-update/angular-location-update.js",

    paths.thirdpartySrc + "ng-table/ng-table.js",

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

    paths.thirdpartySrc + "rison-node/rison.js",
    paths.thirdpartySrc + "auth0-js/auth0.js",
    paths.thirdpartySrc + "angular-auth0/angular-auth0.js",
];

// Enumerate the static assets (including thirdparty.js) so that the RPM/DEB
//      packages only install what we really need
var staticFiles = [
    paths.staticSrc + '*.*',
    paths.staticSrc + 'css/**/*.*',
    paths.staticSrc + 'doc/**/*.*',
    paths.staticSrc + 'fonts/**/*.*',
    paths.staticSrc + 'help/**/*.*',
    paths.staticSrc + 'i18n/**/*.*',
    paths.staticSrc + 'ico/**/*.*',
    paths.staticSrc + 'img/**/*.*',
    paths.staticSrc + 'lib/bootstrap/dist/**/*.*',
    paths.staticSrc + 'lib/codemirror/lib/*.css',
    paths.staticSrc + 'lib/jquery-ui/themes/base/*.*',
    paths.staticSrc + 'lib/jquery-datetimepicker/*.css',
    paths.staticSrc + 'lib/babel-polyfill/*',
    paths.staticSrc + 'lib/thirdparty.*',
    paths.staticSrc + 'scripts/**/*.*',
    paths.staticSrc + 'lib/ng-table/ng-table.css'
];

// set to true to skip js transpile
// NOTE: this is set to true automatically
// when running the "watch" task
var fastBuild = false;

module.exports = {
    paths,
    controlplaneFiles,
    controlplanePartials,
    thirdpartyFiles,
    staticFiles,
    fastBuild
};
