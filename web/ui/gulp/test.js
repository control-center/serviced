/* jshint node: true */

var gulp = require("gulp"),
    gutil = require('gulp-util'),
    karma = require('karma'),
    path = require("path");

/*
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
*/


// headlessly runs unit test suite using chrome and xvfb
gulp.task('test', function (done) {
    // TODO - find chrome
    process.env.CHROME_BIN = "/usr/bin/google-chrome";
    // TODO - find an available display: test -e /tmp/.X99-lock
    // TODO - kick off xvfb at that display: /usr/bin/Xvfb :99
    process.env.DISPLAY = ":99.0";
    
    let server = new karma.Server({
        configFile: path.resolve("karma.conf.js"),
        singleRun: true,
        logLevel: "debug",
        browsers: ["Chrome"],
        reporters: ["progress","junit","coverage","threshold"],
    }, function(exitStatus) {
        // Workaround for 'formatError' based on suggestions from
        //   http://stackoverflow.com/questions/26614738/issue-running-karma-task-from-gulp
        // but tweaked to use (apparently new) PluginError
        var err = exitStatus ? new gutil.PluginError('test', 'There are failing unit tests') : undefined;
        done(err);
    });

    server.start();
});

// brings up web browser and auto-runs tests as they
// are saved and edited
gulp.task('tdd', function (done) {
    let server = new karma.Server({
        configFile: __dirname + '/karma.conf.js',
        browsers: ["Chrome"],
        reporters: ["html"],
        autoWatch: true
    }, function(exitStatus) {
        // Workaround for 'formatError' based on suggestions from
        //   http://stackoverflow.com/questions/26614738/issue-running-karma-task-from-gulp
        // but tweaked to use (apparently new) PluginError
        var err = exitStatus ? new gutil.PluginError('test', 'There are failing unit tests') : undefined;
        done(err);
    });
    server.start();
});


