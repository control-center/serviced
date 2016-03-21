# Automated Acceptance Tests


The `ui` subdirectory contains UI functional tests which can be executed in a completely automated fashion using [Capybara](https://github.com/jnicklas/capybara) in a [Docker](http://www.docker.com) container.

The tests may be run against Firefox, Chrome, or Poltergeist/Phantomjs. It also includes support for [screenshots](https://github.com/mattheworiordan/capybara-screenshot) of failed tests cases.

#Table of Contents

  - [Overview](#overview)
  - [How to run](#how-to-run)
    - [Step 1 - Start Control Center](#step-1---start-control-center)
    - [Step 2 - Build the docker image (optional)](#step-2---build-the-docker-image)
    - [Step 3 - Setup the test environment](#step-3---setup-the-test-environment)
    - [Step 4 - Run the test suite](#step-4---run-the-test-suite)
    - [Step 5 - Review the test results](#step-5---review-the-test-results)
    - [Cucumber Command Line Options](#cucumber-command-line-options)
    - [Environment variables](#environment-variables)
  - [Examples](#examples)
    - [Running a subset of tests](#running-a-subset-of-tests)
    - [Looking at a failed test case](#looking-at-a-failed-test-case)
    - [Tagging conventions](#tagging-conventions)
    - [Watching the browser while the tests run](#watching-the-browser-while-the-tests-run)
    - [Mark a test PENDING](#mark-a-test-pending)
    - [Page Object Model](#page-object-model)
  - [Guidelines](#guidelines)
  - [TODOs](#todos)
  - [Known Issues](#known-issues)
  - [References](#references)

## Overview

The docker image `zenoss/capybara` contains all of the tools and libraries required to run Capybara against Firefox, Chrome or Phantomjs.

The subdirectory `ui` is mounted into the docker container under the directory `/capybara`, giving the tools in the container access to all of the cucumber/capybara test files defined in `ui`.

The script `runCucumber.sh` is executed from within docker. It handles any runtime setup (such as starting Xvfb if necessary), and
then executes Cucumber.

A report of the test execution is written to `ui/output`.

## How to run

### Step 1 - Start Control Center
The test suite assumes serviced is already running and ready to receive requests
from a web browser.

To run the test, you must specify the URL for Control Center either
by setting the environment variable APPLICATION_URL or by specifying the
command line argument `-a <url>`. Do not use https://localhost because from
the perspective of Capybara/Cucumber running inside the container, https://localhost
refers to the container itself.

### Step 2 - Build the docker image (optional)
The docker image containing Cucumber and Capybara is available on the Zenoss dockerhub repo as `zenoss/capybara:$(VERSION)`, so you do not need to build it yourself.

In case you do want/need to build the docker image containing Cucumber and Capybara, use the following commands:

```
$ zendev cd serviced
$ cd acceptance/dockerImage
$ make
```

Once the image has been built, you can use the following commands to push it to Docker hub:

```
$ zendev cd serviced
$ cd acceptance/dockerImage
$ make dockerPush
```

**NOTE:** The version of the docker image is defined by the file [dockerImage/VERSION](dockerImage/VERSION).
If you modify the contents of the image, you should update the version number in that file before
building/pushing a new image.

### Step 3 - Setup the test environment

#### Start the mock agents
The test suite uses mock agents for tests involving hosts. The mock agents do not (currently) connect to Zookeeper, nor do they have the ability to mock service executions. Their primary purpose is to provide light-weight mocks for the set of properties for each host (CPU, RAM, Kernel Version, CC version, etc). To build the mock agent, use the following commands:

```
$ zendev cd serviced
$ make mockAgent
```

To run the mock agents, use the following commands:

```
$ cd acceptance
$ ./startMockAgents.sh
```

This step is not necessary if you do not run tests involving hosts.

**NOTE:** If you stop and restart the start mock agents script while serviced is running, you may see a "Bad Request: connection is shut down" error when you try to add a mock agent. Restarting serviced will fix this.

#### Add the test template
The Application tests assume that a test template has already been added to the system. To comple and add the test template, use the following commands:

```
$ zendev cd serviced
$ serviced template compile dao/testsvc | serviced template add
```

In the future, this step may be incorporated directly into the Cucumber tests for Applications.

### Step 4 - Run the test suite

Capybara uses different 'drivers' to interface with a web browser.
By default, `runUIAcceptance.sh` executes tests against Chrome.
The test suite can be run against any one of several browsers by selecting different drivers.
Both of the following commands run the test suite against Chrome:

```
$ ./runUIAcceptance.sh -a <servicedURL> -u <userID> -p <password>
or
$ ./runUIAcceptance.sh -d selenium_chrome -a <servicedURL> -u <userID> -p <password>
```

To run the tests against Firefox, use

```
$ ./runUIAcceptance.sh -d selenium -a <servicedURL> -u <userID> -p <password>
```

To run the tests against Poltergeist/Phantomjs, use

```
$ ./runUIAcceptance.sh -d poltergeist -a <servicedURL> -u <userID> -p <password>
```

For a full description of the command line options, run `./runUIAcceptance.sh -h`

### Step 5 - Review the test results

The output from the tests are written to stdout as the tests execute. Additionally, an HTML report is written to
`ui/output/feature-overview.html`. If a test case fails, the HTML report will include a screenshot of the browser
at the point in time when the test case failed.

### Cucumber Command Line Options
Cucumber command line options can be specified by defining the environment variable `CUCUMBER_OPTS` on the make command line.

### Environment variables
Environment variables are used to pass information into the docker container which are used by the shell and Ruby scripts executed in the container.

The primary variables used by `runUIAcceptance.sh` are:

 * **`APPLICATION_URL`** - the URL of the application under test. You can set this variable with the `-a` command line option for `runUIAcceptance.sh`.
 * **`APPLICATION_USERID`** - the user id to login into the application under test. You can set this variable with the `-u` command line option for `runUIAcceptance.sh`.
 * **`APPLICATION_PASSWORD`** - the password used to login into the application under test. You can set this variable with the `-p` command line option for `runUIAcceptance.sh`.
 * **`CAPYBARA_DRIVER`** - the name of the Capybara web driver to use. Valid values are `selenium` (which uses Firefox), `selenium_chrome`, or `poltergeist` (which uses PhantomJS). The default if not specified is `selenium_chrome`. You can set this variable with the `-d` command line option for `runUIAcceptance.sh`.
 * **`CAPYBARA_TIMEOUT`** - the timeout, in seconds, that Capybara should wait for a page or element. The default is 10 seconds. You can set this variable with the `-t` command line option for `runUIAcceptance.sh`.
 * **`CUCUMBER_OPTS`** - any of the standard command line options for Cucumber.
 * **`DATASET`** - the JSON dataset to use as test input. You can set this variable with the `--dataset` command line option for `runUIAcceptance.sh`.

Internally, the script also uses the variables `CALLER_UID` and `CALLER_GID` to capture the current user's UID and GID which are used in the container so that
files written to `ui/output` will have the proper owner/group information (for OSX, see Known Issues). These two variables should not be overwritten or modified. For an example of how these two variables are used, refer to the [dockerImage/build/runCucumber.sh](dockerImage/build/runCucumber.sh) script.

For details of how all of these variables are used, see [ui/features/support/env.rb](ui/features/support/env.rb) and [ui/features/support/application.rb](ui/features/support/application.rb)

For a full list of possible options for Cucumber itself, pass the `--help` option to Cucumber like this:

```
$ CUCUMBER_OPTS=--help ./runUIAcceptance.sh -a <servicedURL> -u <userid> -p <password>
```

## Examples
### Running a subset of tests
Cucumber supports a feature called tags which can be used to run a subset of tests.  A full explanation of using tags is beyond the scope of this document; see the Cucumber [documentation](https://github.com/cucumber/cucumber/wiki/Tags) for a full description.

For example, you can run tests for a single tag with a command like:

```
$ ./runUIAcceptance.sh -a <servicedURL> -u <userid> -p <password> --tags @hosts

```

or you can include tags as part of the `CUCUMBER_OPTS` environment variable

```
$ CUCUMBER_OPTS='--tags @hosts' ./runUIAcceptance.sh -a <servicedURL> -u <userid> -p <password>

```

### Looking at a failed test case
If a test step fails, the test harness will capture a screenshot. If you have not seen a failed test case already, run the tests with an invalid userid and/or password.  If you open `ui/output/feature-overview.html` with a browser and drill into the details for the login feature, you will see the output in a nicely formatted table. Look for a "Screenshot" link below the error report for the failed step. Clicking on that link should display an image captured at the time of the failure.

### Tagging conventions
Cucumber offers a powerful tagging feature that can be used to control which features and/or scenarios are run, as well as enabling custom 'hooks' to run specific blocks of code at different points.

Some of the tags defined by this project are:

 * feature tags - There is a unique tag on each Feature to allow running single features at a time or in combination. Some of the valid values are `@login`, and `@hosts`
 * login hook - The tag `@login-required` illustrates how to use hook tags to automatically execute some block of code before/after the associated feature/step. In this case, the `@login-required` tag will login the user before each feature/step decorated with the tag (remember that each Scenario executes as a new browser session).

 To specify one or more of these tags, add the `--tags tagName` to the command line. For instance, the following command will run just the tests for the hosts feature:

 ```
$ ./runUIAcceptance.sh -a servicedURL -u yourUserID -p yourPasswordHere --tags @hosts
 ```

For information about this Cucumber feature, see:

 * [Tags](https://github.com/cucumber/cucumber/wiki/Tags)
 * [Hooks](https://github.com/cucumber/cucumber/wiki/Hooks)

### Watching the browser while the tests run
Normally, the browser used by the tests is executed within the Docker container using
the X virtual framebuffer (Xvfb) so nothing is displayed.  For debugging purposes,
it is often useful to view the browser while the test runs.  To do that, include the `--debug`
argument as illustrated below:

```
$ xhost +
$ ./runUIAcceptance.sh --debug -a servicedURL -u yourUserID -p yourPasswordHere
```

*NOTE:* You only need to execute the command `xhost +` once.

### Mark a test PENDING
When a test is marked `PENDING`, Cucumber will skip it. This is useful for several scenarios, including
Behavior Driven Development (BDD), where you want to first write the test case, and then write the
code to make the test case pass. However, in between
the time you create the test case and the time you implement the code, you may not want all of
the intervening test-case executions to fail (e.g. tests launched by an automated build).
The following example illustrates how to mark a test case as PENDING:

```
  @login-required
  Scenario: View empty Hosts page
    When PENDING I am on the hosts page
    Then I should see "Applications"
      And I should see "Hosts Map"
      And I should see "No Data Found"
      And I should see "Showing 0 Results"
```

### Page Object Model
The combination of Cucumber and Capybara offer huge advantages in terms of being able to write tests using a simple, expressive DSL that describes how to interact with your application's UI.
However, as your application and your tests grow, you can easily run into situations where implementation details about a particular page or reusable element are either 'leaking' into Step statements explicitly, or they are constantly repeated across step definitions. For instances, things like the IDs or CSS/Xpath expressions used to identify specific elements on a page can end up being repeated over and over again. When the actual page definition is changed by a developer, then you have to make the same refactor across multiple places in your tests.

A Page Object Model is a DSL for describing for the page itself. Think of it as a secondary DSL "below" the DSL expresssed in the test features. The Page Object Model should encapsulate all of the implementation detail like DOM identifiers, CSS/xpath matching expressions, etc.

These tests use [Site Prism](https://github.com/natritmeyer/site_prism) to implement a Page Object Model.  The various page objects are is defined in the [ui/features/pages](ui/features/pages) directory and used in the step definitions. For instance, the login page is defined in [ui/features/pages/login.rb](ui/features/pages/login.rb), and it is used in [ui/features/steps/login_steps.rb](ui/features/steps/login_steps.rb).

For more discussion of the Page Object Model, see

 * [Testing Page Objects with SitePrism](http://www.sitepoint.com/testing-page-objects-siteprism/)
 * [Keeping It Dry With Page Objects](http://techblog.constantcontact.com/software-development/keeping-it-dry-with-page-objects/)


## Guidelines
Please follow these [guidelines](Guidelines.md) when writing or modifying tests.

## TODOs

 * Add example of REST validation using Cucumber. Here are some helper libraries to consider:
   * https://github.com/DigitalInnovation/cucumber_rest_api
   * https://github.com/jayzes/cucumber-api-steps
   * https://github.com/collectiveidea/json_spec
 * Add example of CLI validation using Cucumber.

## Known Issues

 * With upgrade to zenoss/capybara:1.1.0, the --debug option stopped working.
 * Phantomjs does not work. The primary issue is lack of support for the ES 6 `bind()` method which also prevents use of PhantomJS with our unit tests.
 * The tests don't work on Mac OSX for a variety of reasons:
   * The run script makes Linux-specific assumptions about mapping timezone definitions into the container.
   * On Mac OSX with [boot2docker](http://boot2docker.io/), if you have problems reaching archive.ubuntu.com while trying to run `dockerBuild.sh`, refer to the workaround [here](http://stackoverflow.com/questions/26424338/docker-daemon-config-file-on-boot2docker).
   * On Mac OSX with [boot2docker](http://boot2docker.io/), boot2docker will automagically map the root user of the docker container to the current user of the host OS. This use-case needs more testing.

## References

 * [Docker](http://www.docker.com)
 * [Cucumber - A tool for BDD testing](https://github.com/cucumber/cucumber/wiki)
 * [The Cucumber Book](https://www.safaribooksonline.com/library/view/the-cucumber-book/9781941222911/) by Aslak Hellesoy, Matt Wynne
 * [Capybara - An Acceptance test framework for web applications](https://github.com/jnicklas/capybara)
 * [Documentation for Capybara](https://www.rubydoc.info/github/jnicklas/capybara/index)
 * [Application Testing with Capybara](https://www.safaribooksonline.com/library/view/application-testing-with/9781783281251/) by Matthew Robbins
 * [Capybara cheat sheet](https://gist.github.com/zhengjia/428105)
 * [Site Prism - A Page Object Model DSL for Capybara](https://github.com/natritmeyer/site_prism)
 * [How to install PhantomJS on Ubuntu](https://gist.github.com/julionc/7476620)
 * [Notes on setting up Xvfb for docker](https://github.com/keyvanfatehi/docker-chrome-xvfb)
 * [How to install Chromedriver on Ubuntu](https://devblog.supportbee.com/2014/10/27/setting-up-cucumber-to-run-with-Chrome-on-Linux/)
 * [How to install Chrome from the command line](http://askubuntu.com/questions/79280/how-to-install-chrome-browser-properly-via-command-line)
 * [URLs for different versions of Chrome](http://www.ubuntuupdates.org/package/google_chrome/stable/main/base/google-chrome-stable)
 * [Chromedriver options](https://sites.google.com/a/chromium.org/chromedriver/capabilities)
 * [Chrome options](http://peter.sh/experiments/chromium-command-line-switches/)
