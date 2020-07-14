/* global module: true */
// Karma configuration
// Generated on Thu Jan 22 2015 09:23:45 GMT-0600 (CST)



module.exports = function(config) {
  config.set({

    // base path that will be used to resolve all patterns (eg. files, exclude)
    basePath: '',


    // frameworks to use
    // available frameworks: https://npmjs.org/browse/keyword/karma-adapter
    frameworks: ['jasmine'],


    // list of files / patterns to load in the browser
    files: [
      'static/lib/thirdparty.js',
      'static/lib/angular/angular-mocks.js',
      'test/appTest.js',
      'test/**/*Mock.js',
      'test/**/*Spec.js',
      'src/**/*.js'
    ],


    // list of files to exclude
    exclude: [
        'test/obsoleteSpec.js'
    ],


    // preprocess matching files before serving them to the browser
    // available preprocessors: https://npmjs.org/browse/keyword/karma-preprocessor
    preprocessors: {
        'src/**/*.js': ['babel', 'coverage']
    },

    bablePreprocessor: {
        options: ["es2015"]
    },

    // test results reporter to use
    // possible values: 'dots', 'progress', 'html', 'junit', 'coverage'
    // available reporters: https://npmjs.org/browse/keyword/karma-reporter
    reporters: ['dots'],


    // web server port
    port: 9876,


    // enable / disable colors in the output (reporters and logs)
    colors: true,


    // level of logging
    // possible values: config.LOG_DISABLE || config.LOG_ERROR || config.LOG_WARN || config.LOG_INFO || config.LOG_DEBUG
    logLevel: config.LOG_INFO,


    // enable / disable watching file and executing tests whenever any file changes
    autoWatch: false,

    customLaunchers: {
      chrome_no_sandbox: {
        base: 'Chrome',
        flags: ['--no-sandbox', '--headless', '--disable-gpu', '--remote-debugging-port=9222']
      }
    },

    // start these browsers
    // available browser launchers: https://npmjs.org/browse/keyword/karma-launcher
    browsers: ['chrome_no_sandbox'],


    // Continuous Integration mode
    // if true, Karma captures browsers, runs the tests and exits
    singleRun: false,


    // Junit-style reports that can be displayed in Jenkins
    // For more info, see https://www.npmjs.com/package/karma-junit-reporter
    junitReporter: {
      outputDir: 'test/results',
      outputFile: 'results.xml',
      useBrowserName: false,
      suite: ''
    },


    // Test coverage configuiraion.
    // For more info, see http://karma-runner.github.io/0.8/config/coverage.html
    coverageReporter: {
      // specify a common output directory
      dir: 'test/results/coverage',
      reporters: [
        { type: 'cobertura', subdir: 'cobertura' }  // for integration with Jenkins
      ]
    },


    // Code coverage results below these thresholds will trigger a build failure
    // For more info, see https://www.npmjs.com/package/karma-threshold-reporter
    thresholdReporter: {
      statements: 17,
      branches: 13,
      functions: 15,
      lines: 17
    }
  });
};
