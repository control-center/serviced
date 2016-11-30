/* jshint node: true */

var gulp = require("gulp"),
    gutil = require('gulp-util'),
    karma = require('karma'),
    path = require("path"),
    which = require("which"),
    child_process = require("child_process");

function setChromePath(){
    try {
        process.env.CHROME_BIN = which.sync("google-chrome");
    } catch(e) {
        return;
    }
   return process.env.CHROME_BIN;
}

function startXvfb(display){
    return child_process.exec(`Xvfb :${display}`);
}

// headlessly runs unit test suite using chrome and xvfb
gulp.task('test', function (done) {
    if(!setChromePath()){
        done(new gutil.PluginError("test", "Could not find google-chrome"));
        return;
    }
    gutil.log(`got chrome path ${process.env.CHROME_BIN}`);

    let display;
    for(display = 99; display > 50; display--){
        // see if this display is available
        let result = child_process.spawnSync("test", ["-e", `/tmp/.X${display}-lock`]);
        // 1 means the file doesnt exists
        if(result.status === 1){
            break;
       }
    }
    gutil.log(`found display ${display}`);
    // gues we didnt find a display
    if(display === 50){
        done(new gutil.PluginError("test", "Could not find a display for Xvfb to use"));
        return;
    }

    process.env.DISPLAY = `:${display}.0`;

    startXvfb(display)
       .on("error", e => {
            // TODO - gracefully kill server if running
            done(new gutil.PluginError("test", `Xvfb failed with ${e}`));
        });

    gutil.log("started Xvfb");

    let server = new karma.Server({
        configFile: path.resolve("karma.conf.js"),
        singleRun: true,
        logLevel: "debug",
        browsers: ["chrome_no_sandbox"],
        reporters: ["progress","junit","coverage","threshold"],
    }, function(exitStatus) {
        var err = exitStatus ? new gutil.PluginError('test', 'There are failing unit tests') : undefined;
        done(err);
        // HACK - karma hangs indefinitely on a singleRun, 
        // so forcefully end the process
        process.exit(exitStatus);
    });

    server.start();
});

// brings up web browser and auto-runs tests as they
// are saved and edited. NOTE: this wont work headless
gulp.task('tdd', function (done) {
    if(!setChromePath()){
        done(new gutil.PluginError("test", "Could not find google-chrome"));
        return;
    }

    let server = new karma.Server({
        configFile: path.resolve('karma.conf.js'),
        browsers: ["Chrome"],
        reporters: ["kjhtml"],
        autoWatch: true
    }, function(exitStatus) {
        var err = exitStatus ? new gutil.PluginError('test', 'There are failing unit tests') : undefined;
        done(err);
    });

    server.on("browser_error", (browser, error) =>{
        done(new gutil.PluginError("test", "Karma experienced a browser error. sorry. its not your fault."));
    });

    server.start();
});


