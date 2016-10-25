/* jshint node: true */

var gulp = require("gulp"),
    gutil = require('gulp-util'),
    karma = require('karma').server;

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


// brings up web browser and auto-runs tests as they
// are saved and edited
gulp.task('tdd', function (done) {
  karma.start({
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
});
