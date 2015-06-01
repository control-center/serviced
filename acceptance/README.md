# Automated Acceptance Tests


The `ui` subdirectory contains UI functional tests which can be executed in a completely automated fashion using [Capybara](https://github.com/jnicklas/capybara) in a [Docker](http://www.docker.com) container.

The tests may be run against Firefox, Chrome, or Poltergeist/Phantomjs. It also includes support for [screenshots](https://github.com/mattheworiordan/capybara-screenshot) of failed tests cases

#Table of Contents

  - [Overview](#overview)
  - [How to run](#how-to-run)
    - [Step 1 - Start Control Center](#step-1---start-control-center)
    - [Step 2 - Run the test suite](#step-2---run-the-test-suite)
    - [Step 3 - Review the test results](#step-3---review-the-test-results)
    - [Cucumber Command Line Options](#cucumber-command-line-options)
    - [Environment variables](#environment-variables)
  - [Examples](#examples)
    - [Running a subset of tests](#running-a-subset-of-tests)
    - [Looking at a failed test case](#looking-at-a-failed-test-case)
    - [Tagging conventions](#tagging-conventions)
    - [Page Object Model](#page-object-model)
  - [TODOs](#todos)
  - [Known Issues](#known-issues)
  - [References](#references)

## Overview

The docker image `zenoss/capybara` contains all of the tools and libraries required to run Capybara against Firefox, Chrome or Phantomjs.

The subdirectory `ui` is mounted into the docker container under the directory `/capybara`, giving the tools in the container access to all of the cucumber/capybara test files defined in `ui`.

The script `runCucmber.sh` is executed from within docker. It handles any runtime setup (such as starting Xvfb if necessary), and
then executes cucumber.

A report of the test execution is written to `ui/output`.

## How to run

### Step 1 - Start Control Center
The test suite assumes serviced is already running and ready to receive requests
from a web browser.

The test runner assumes that CC is reachable via https://localhost. If Control Center
is running on another machine, you can tell the runner the URL of that machine either
by setting the environament variable APPLICATION_URL or specifying `-a <url>`
command line argument.

### Step 2 - Run the test suite

Capybara uses different 'drivers' to interface with a web browser.
The Selenium driver for Capybara is the default, and by default it executes tests againts Firefox.
The test suite can be run against any one of several browsers by selecting differnet drivers.

```
$ ./runUIAcceptance.sh -u <userID> -p <password>
or
$ ./runUIAcceptance.sh -d selenium -u <userID> -p <password>
```

To run against Chrome,

```
$ ./runUIAcceptance.sh -d selenium_chrome -u <userID> -p <password>
```

To run against Poltergeist/Phantomjs,

```
$ ./runUIAcceptance.sh -d poltergeist -u <userID> -p <password>
```

For a full description of the command line options, run `./runUIAcceptance.sh -h`

### Step 3 - Review the test results

The output from the tests are written to stdout as the tests execute. Additionally, an HTML report is written to
`ui/output/feature-overview.html`. If a test case fails, the HTML report will include a screenshot of the browser
at the point in time when the test case failed.

### Cucumber Command Line Options
Cucumber command line options can be specified by defining the environment variable `CUCUMBER_OPTS` on the make command line.

### Environment variables
Environment variables are used to pass information into the docker container which are used by the shell and Ruby scripts executed in the container.

The primary variables used by `runUIAcceptance.sh` are:

 * **`APPLICATION_URL`** - the URL of the application under test (defaults to https://localhost).
 * **`APPLICATION_USERID`** - the user id to login into the application under test. You can set this variable with the `-u` command line option for `runUIAcceptance.sh`.
 * **`APPLICATION_PASSWORD`** - the password used to login into the application under test. You can set this variable with the `-p` command line option for `runUIAcceptance.sh`.
 * **`CAPYBARA_DRIVER`** - the name of the Capybara web driver to use. Valid values are `selenium` (which uses Firefox), `selenium_chrome`, or `poltergeist` (which uses PhantomJS). The default if not specified is "selenium". You can set this variable with the `-d` command line option for `runUIAcceptance.sh`.
 * **`CAPYBARA_TIMEOUT`** - the timeout, in seconds, that Capybara should wait for a page or element. The default is 10 seconds. You can set this variable with the `-t` command line option for `runUIAcceptance.sh`.
 * **`CUCUMBER_OPTS`** - any of the standard command line options for Cucumber.

Internally, the script also uses the variables `CALLER_UID` and `CALLER_GID` capture the current users UID and GID which are used in the container to create a `cuke` user so that
files written to `ui/output` will have the proper owner/group information (for OSX, see Known Issues). These two variables should not be overwritten or modified. For an example of how these variables are used, refer to the [dockerImage/build/makeCukeUser.sh](dockerImage/build/makeCukeUser.sh) script.

For details of how these variables are used, see [ui/features/support/env.rb](ui/features/support/env.rb) and [ui/features/support/application.rb](ui/features/support/application.rb)

For a full list of possible options for Cucumber itself, pass the `--help` option to Cucumber like this:

```
$ CUCUMBER_OPTS=--help ./runUIAcceptance.sh
```

## Examples
### Running a subset of tests
Cucumber supports a feature called tags which can be used in run a subset of tests.  A full explanation of using tags is beyond the scope of this document; see the Cucumber [documentation](https://github.com/cucumber/cucumber/wiki/Tags) for a full description.

For example, you can run tests for a single tag with a command like:

```
$ CUCUMBER_OPTS='--tags @hosts' ./runUIAcceptance.sh -u <userid> -p <password>

```

### Looking at a failed test case
If a test step fails, the test harness will capture a screenshot. If you have not seen a failed test case already, run the tests with an invalid userid and/or password.  If you open `ui/output/feature-overview.html` with a browser and drill into the details for the login feature, you will see the output in nicely formatted table. Look for a "Screenshot" link below the error report for the failed step. Clicking on that link should display an image captured at the time of the failure.

### Tagging conventions
Cucumber offers a powerful tagging feature that can be used to control which features and/or scenarios are run, as well as enabling custom 'hooks' to run specific blocks of code at different points

Some of the tags defined by this project are:

 * feature tags - There is a unique tag on each Feature to allow running single features at a time or in combination. Some of the valid values are `@login`, and `@hosts`
 * login hook - The tag `@login-required` illustrates how to use hook tags to automatically execute some block of code before/after the associated feature/step. In this case, the `@login-required` tag will login the user before each feature/step decorated with the tag (remember that each Scenario executes as a new browser session).

 To specify one of these tags, define `--tags tagName` in CUCUMBER_OPTS. For instance the following command will run just the tests for the hosts feature:

 ```
$ CUCUMBER_OPTS='--tags @hosts' ./runUIAcceptance.sh -u yourName@something.com -p yourPasswordHere`
 ```

For information of these Cucumber feature, see:

 * [Tags](https://github.com/cucumber/cucumber/wiki/Tags)
 * [Hooks](https://github.com/cucumber/cucumber/wiki/Hooks)

### Page Object Model
Cucmber and Capybara offer huge advantages in terms of being able to write tests using a simple, expressive DSL that describes how to interact with your application's UI.
However, as your application and your tests grow, you can easily run into situations where implementation details about a particular page or reusable element are either 'leaking' into Step statements explicitly, or they are constantly repeated across step definitions. For instances, things like the IDs or CSS/Xpath expressions used to identify specific elements on a page can end up being repeated over and over again. When the actual page definition is changed by a developer, then you have to make the same refactor across multiple places in your tests.

A Page Object Model is a DSL for describing for the page itself. Think of it as a secondary DSL "below" the DSL expresssed in the test features. The Page Object Model should encapsulate all of the implementation detail like DOM identifiers, CSS/xpath matching expressions, etc.

These tests use [Site Prism](https://github.com/natritmeyer/site_prism) to implement a Page Object Model.  The various page objects are is defined in the [ui/features/pages] directory and used in the step definitions. For instance, the login page is defined in [ui/features/pages/login.rb](ui/features/pages/login.rb), and it is used in [ui/features/steps/login_steps.rb](ui/features/steps/login_steps.rb).

For more discussion of page object model, see

 * [Testing Page Objects with SitePrism](http://www.sitepoint.com/testing-page-objects-siteprism/)
 * [Keeping It Dry With Page Objects](http://techblog.constantcontact.com/software-development/keeping-it-dry-with-page-objects/)

## TODOs

 * Add example of REST validation

## Known Issues

 * Phantomjs does not work. The primary issue is lack of support for the ES 6 `bind()` method which also prevents use of PhantomJS with our unit tests.
 * The tests don't work on Mac OSX for a variety of reasons:
   * The run script makes Linux-specific assumptions about mapping timezone definitions into the container.
   * On Mac OSX with [boot2docker](http://boot2docker.io/), if you have problems reaching archive.ubuntu.com while trying to run `dockerBuild.sh`, refer to the workaround [here](http://stackoverflow.com/questions/26424338/docker-daemon-config-file-on-boot2docker).
   * On Mac OSX with [boot2docker](http://boot2docker.io/), you must use the `--root` option for `runUIAcceptance.sh` and
 boot2docker will automagically map the root user of the docker container to the current user of the host OS. Without the `--root` option, you will encounter permission problems trying to write files into the `ui/output` directory.

## References

 * [Docker](http://www.docker.com)
 * [Cucumber - A tool for BDD testing](https://github.com/cucumber/cucumber)
 * [Capybara - An Acceptance test framework for web applications](https://github.com/jnicklas/capybara)
 * [Capybara cheat sheet](https://gist.github.com/zhengjia/428105)
 * [Site Prism - A Page Object Model DSL for Capybara](https://github.com/natritmeyer/site_prism)
 * [How to install PhantomJS on Ubuntu](https://gist.github.com/julionc/7476620)
 * [Notes on setting up Xvfb for docker](https://github.com/keyvanfatehi/docker-chrome-xvfb)
 * [How to install Chromedriver on Ubuntu](https://devblog.supportbee.com/2014/10/27/setting-up-cucumber-to-run-with-Chrome-on-Linux/)
 * [How to install Chrome from the command line](http://askubuntu.com/questions/79280/how-to-install-chrome-browser-properly-via-command-line)
 * [URLs for different versions of Chrome](http://www.ubuntuupdates.org/package/google_chrome/stable/main/base/google-chrome-stable)
 * [Chromedriver options](https://sites.google.com/a/chromium.org/chromedriver/capabilities)
 * [Chrome options](http://peter.sh/experiments/chromium-command-line-switches/)
