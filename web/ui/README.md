# serviced/web/ui

The `serviced/web/ui` folder contains the UI code for the serviced.

## Directory structure
The following is a partial illustration of the subdirectory structure for the parent directory `serviced/web`
 ```
 + serviced
   |
   +-- web               REST backend source
   +-- ui                UI source
       |
 	     + static
         |
         +-- build       build results including the full set of files for runtime
         |
         +-- src         UI javascript code
         |
         +-- static      static assets used by the UI including thirdparty libraries
         |
         +-- test        unit-test code for the UI javascript
 ```

## Build Setup
Nothing special is required to build the code in `serviced/web`. The toplevel makefile for serviced calls the `serviced/web/ui/makefile`to build the UI code By default, the `serviced/web/ui/makefile` uses the Docker container `zenoss/serviced-build` defined in `serviced/build/Dockerfile` to launch the UI portion of the build. This image contains all of the tools required to build the UI.

If you have node.js installed locally, then the first time the build is executed a number of additional third-party UI build tools will be automatically downloaded and cached locally. This may take a few minutes. However, after the tools are cached locally, subsequent builds will not repeat those downloads.

If you do not have node.js installed locally, these tools are included in the Docker image `zenoss/serviced-build`.

### Primary make targets
The three primary make targets are `build`, `test`, and `clean`. All of these targets perform the corresponding action on the Javascript code. Developers who want to build/test/clean _only_ the Go code or _only_ the Javascript code should use the native build tools for those languages directly rather than make. For Javascript, the primary build tool is [gulp](http://gulpjs.com/).

### Installing dev tools locally

Do these things and then you'll be able to develop the CC UI app supes fast!

install nodejs 6

    curl -sL https://deb.nodesource.com/setup_6.x | sudo -E bash -
    sudo apt-get install -y nodejs

make sure you have build tools (for native npm modules)

    sudo apt-get install -y build-essential

install a few global npm packages

    sudo npm install --global gulp jshint babel yarn

navigate to the `web/ui` directory and remove `node_modules` if present

    rm -rf node_modules

install CC UI dev dependencies using yarn

    yarn install --pure-lockfile

use our local npm repo by adding the following line to your `$HOME/.npmrc`:

    registry = http://nexus.zenoss.eng:8081/nexus/content/repositories/npm

And now you're all set to develop the CC UI app locally. The previous steps are all (mostly) one-time setup. To develop the web app, open either Chrome (recommended) or Firefox's dev tools, make sure caching is off (you should be able to specifically disable caching when dev tools is open), and run the following in a terminal:

    gulp watch

Now whenever changes are made to the source html, css, or js, the web app will be rebuilt on the fly. Refresh the page in the browser and you will see your changes right away. Boom! Now learn to use Chrome's dev tools! They're amazing!

NOTE: `gulp watch` will not transpile javascript, which means you need to keep your browser up to date to use this feature. If this is not possible, manually run `gulp` each time a change is made to the javascript source and gulp will build AND transpile the code to work on all supported browsers (IE10+).

### Installing dev tools locally - the longer version
It is recommended (but not required) that developers working on the UI code install the Javascript build tools directly.
The makefile will not use the Docker container `zenoss/serviced-build` if it finds [Node.js](http://nodejs.org)
on the user's path. Therefore, bypassing the Docker container will speed up your local builds a little bit.

If you are building locally, please create (or update) your `$HOME/.nmprc` file to contain the following line:
```
registry = http://nexus.zenoss.eng:8081/nexus/content/repositories/npm
```
This will direct NPM to pull artifacts from the Zenoss-local NPM repo on our
[Nexus server](http://www.sonatype.com/nexus-repository-sonatype) -
http://nexus.zenoss.eng:8081/nexus/#view-repositories

Regardless of whether you are using the `zenoss/serviced-build` container or installing the tools locally, the tool chain for UI builds is divided into two parts:
  * pre-requisite tools
  * all other build-time dependencies

The pre-requisite tools are ones which must be pre-installed. The `zenoss/serviced-build` container includes these tools.
If you want to install them locally, refer to the commands in [`serviced/build/Dockerfile`](../../build/Dockerfile) for the following packages:
  * [Node.js](http://nodejs.org) - a Javascript application platform
  * [npm](https://www.npmjs.com/) - the node package manager. npm is bundled with the Windows and Mac distros of node.js, but has to be installed separately for Linux
  * [yarn](https://yarnpkg.com/) - an alternate package manager that provides reproducible builds and quicker dependency resolution
  * [gulp](http://gulpjs.com/) - a Javascript build tool
  * [babeljs](https://babeljs.io/) - a Javascript transpiler that converts modern js into more broadly compatible js

Once the pre-requisite build tools are installed, all other components of the JS tool chain are downloaded by npm based on the definitions in [`serviced/web/ui/package.json`](./package.json).  If you build with make, this download happens automatically. If you are not building with make, you will need to run the command `yarn install --pure-lockfile` once to download the rest of the tool chain.

**NOTE:** yarn will cache everything it downloads in `serviced/web/ui/node_modules`.  In the unlikely event, you encounter a problem with
incompatible tool versions, you may have to delete this directory and download a fresh set of dependencies by rerunning the make (or running `yarn install --pure-lockfile` if you have installed yarn on your local).


### Updating dev tool versions
To change a version of one of the prerequisite tools (node.js, npm, yarn, gulp or babeljs), you must edit [`serviced/build/Dockerfile`](../../build/Dockerfile) to include the necessary changes.
Be sure that your `$HOME/.npmrc` file is pointed at `http://nexus.zenoss.eng:8081/nexus/content/repositories/npm`
Be sure to test with a clean build, removing `serviced/web/ui/node_modules` just to be safe.

To change a version of one of the other tools, edit [`serviced/web/ui/package.json`](./package.json).

**If a change is made to `serviced/web/ui/package.json`, `yarn.lock` *must* be updated as well.** Use the following procedure to ensure newly installed dependencies are locked down:

```
$ cd web/ui
$ rm -rf node_modules

<< make your changes to package.json >>

$ yarn install

<< build/test with the new changes>>
<< commit changes to package.json and yarn.lock >>
```

Verify a local build works with your changes. Assuming it does, then you need to refresh the `zenoss/serviced-build`
Docker image to include your changes so everyone who does NOT have node.js installed will use them also.

```
$ cd ../..
$ make buildServicedBuildImage
$ make pushServicedBuildImage
```

## Rebuilding thirdparty.js
All of the thirdparty JS libraries used by Control Center are concatenated and minififed into a single file, `serviced/web/ui/static/lib/thirdparty.js`.  Since the concatenation/minification process takes ~10 seconds and the set of third-party libraries does not change very often, the thirdparty.js file is NOT rebuilt as part of the regular build process.
This file must be constructed manually anytime a third-party JS library is added, changed or removed.

```
<< update the library/libraries in serviced/web/ui/static/thirdparty as necessary >>

$ cd web/ui
$ gulp release3rdparty

<< test your changes >>
<< commit the changes to servcied/web/ui/static/thirdparty/* >>
```

## Adding static files to the build image
The JS build stages all necessary runtime files in `serviced/web/ui/build`. Make targets in [`serviced/makefile`](../../makefile) are responsible for copying the entire contents of `serviced/web/ui/build` into the RPM and Debian packages.  If you want to add/change/remove the files delivered in the runtime packages, you must modify the JS build to only stage the files you want. The gulp target `copyStatic` performs the actual copy operation. See the list of files in the variable `staticFiles` defined in [`gulpfile.js`](gulpfile.js).

## Unit-testing
Control Center uses [Jasmine](http://jasmine.github.io/) as the unit-test framework and [Karma](http://karma-runner.github.io/) as the test runner. The Javascript unit-tests are run automatically as part of the CI build (i.e. they are run by `make test`).
To run the tests manually for debugging, use the following steps (all of which assume that the pwd is `serviced/web/ui`)

1. Build your JS changes
The default gulp target will lint, compile and concatenate the code in `serviced/web/ui/src`. The default target will NOT minify the results (`gulp release` includes minification for production builds).

  ```
  $ cd web/ui
  $ gulp
  ```

1. Run the tests

  ```
  gulp test
  ```

1. Launch the unit-tests in a browser for debugging
The `test` target in gulp runs against PhantomJS, a headless webkit browser, which is great for CI builds, but not useful for debugging locally. The following command uses Karma to assemble the necessry test files, and launch them in Chrome for debugging.

  ```
  $ zendev use europa
  $ zendev cd serviced
  $ cd web/ui
  $ ./node_modules/karma/bin/karma start karma.conf.js --browsers Chrome --reporters html --auto-watch
  ```

Once the new browser window is open, click the `DEBUG` button to launch the tests in a new tab. From the new tab, you can open the Developer Tools, set a breakpoint where you like, and refresh the page to rerun the tests.  By specifying, `--auto-watch` on the command line you are telling Karma to watch for file changes, so you can edit your code, save it and refresh the browser to test your latest changes.

To test in Firefox, use the same command but replace "Chrome" with "Firefox".

Here is the command that will run the unit-tests against each of 3 browsers and stop as soon as they are done:

  ```
  $ node_modules/karma/bin/karma start karma.conf.js --browsers Chrome,Firefox,PhantomJS --single-run
  ```

A full explanation of the Karma configuration options is beyond the scope of this README. See the [Karma doc](http://karma-runner.github.io/) for a complete description.

**NOTE:** For Jenkins integration, `gulp test` creates a Junit-style test report in `serviced/web/ui/test/results/results.xml`

## Code Coverage
The default configuration for `gulp test` includes calculation of code coverage statistics. These statistics can be viewed in Jenkins or locally. To view them locally, open the file `serviced/web/ui/test/results/coverage/PhantomJS 1.9.8 (Linux)/index.html`.

The code-coverage configuration includes threshold checking, such that if the coverage falls below certain thresholds, the build will fail.  For the exact threshold values, see the `thresholdReporter` section of [`serviced/web/ui/karma.conf.js`](./karma.conf.js).

**NOTE:** For Jenkins integration, `gulp test` creates a Cobertura-style test report in `serviced/web/ui/test/results/test/results/coverage/PhantomJS 1.9.8 (Linux)/cobertura-coverage.xml`
