FROM ubuntu:trusty
MAINTAINER Zenoss, Inc <dev@zenoss.com>

# Get the basic set of dev tools
RUN	apt-get update -qq && apt-get install -y -q build-essential wget curl unzip make

#
# Install native prerequisites for the ruby gems we need (e.g.nokogiri)
#
RUN apt-get update -qq && apt-get install -y -q zlib1g-dev=1:1.2.8.dfsg-1ubuntu1 libxml2-dev=2.9.1+dfsg1-3ubuntu4.4 libxml2 libxslt1-dev

#
# Install Ruby, Cucumber, Capybara and the other gems we need
#
RUN apt-get update -qq && apt-get install -y -q ruby-full=1:1.9.3.4
RUN gem install cucumber -v 1.3.19
RUN gem install nokogiri -v 1.6.6.2
RUN gem install capybara -v 2.4.4
RUN gem install capybara-screenshot -v 1.0.5
RUN gem install poltergeist -v 1.6.0
RUN gem install rspec -v 3.2.0
RUN gem install selenium-webdriver -v 2.45.0
RUN gem install site_prism -v 2.6

#
# Install the xvfb for firefox and chrome so they can run on a headless system
#
RUN apt-get update -qq && apt-get install -y -q xvfb

#
# Install phantomjs (from https://gist.github.com/julionc/7476620)
#
RUN apt-get update -qq && apt-get install -y -q build-essential chrpath libssl-dev libxft-dev
RUN apt-get update -qq && apt-get install -y -q libfreetype6 libfreetype6-dev
RUN apt-get update -qq && apt-get install -y -q libfontconfig1 libfontconfig1-dev
RUN wget https://bitbucket.org/ariya/phantomjs/downloads/phantomjs-1.9.7-linux-x86_64.tar.bz2 -P /usr/local/share
RUN cd /usr/local/share && tar xvjf phantomjs-1.9.7-linux-x86_64.tar.bz2
RUN ln -sf /usr/local/share/phantomjs-1.9.7-linux-x86_64/bin/phantomjs /usr/local/share/phantomjs
RUN ln -sf /usr/local/share/phantomjs-1.9.7-linux-x86_64/bin/phantomjs /usr/local/bin/phantomjs
RUN ln -sf /usr/local/share/phantomjs-1.9.7-linux-x86_64/bin/phantomjs /usr/bin/phantomjs
RUN rm -rf /usr/local/share/phantomjs-1.9.7-linux-x86_64.tar.bz2

#
# Install firefox.
# Note that the selenium driver doesn't work with all FF versions, so
# if you use a different version of FF, you might have to upgrade selenium or vice-versa
#
RUN wget https://download-installer.cdn.mozilla.net/pub/firefox/releases/36.0/linux-x86_64/en-US/firefox-36.0.tar.bz2 -P /usr/local/share
RUN cd /usr/local/share && tar xvjf firefox-36.0.tar.bz2
RUN ln -fs /usr/local/share/firefox/firefox /usr/bin/firefox
RUN rm -rf /usr/local/share/firefox-36.0.tar.bz2

#
# Install chromedriver that selenium needs to work with chrome
# (from https://devblog.supportbee.com/2014/10/27/setting-up-cucumber-to-run-with-Chrome-on-Linux/)
#
RUN wget -N http://chromedriver.storage.googleapis.com/2.10/chromedriver_linux64.zip -P /tmp
RUN unzip /tmp/chromedriver_linux64.zip -d /tmp
RUN mv /tmp/chromedriver /usr/bin
RUN chmod +x /usr/bin/chromedriver
RUN rm /tmp/chromedriver_linux64.zip

#
# Install chrome - blend of info from several sources
# General process info: http://askubuntu.com/questions/79280/how-to-install-chrome-browser-properly-via-command-line
# Public Key info for safe download: http://www.google.com/linuxrepositories/
# Info about differnet Chrome versions: http://www.ubuntuupdates.org/package/google_chrome/stable/main/base/google-chrome-stable
#
RUN apt-get update -qq && apt-get install -y -q libxss1 libappindicator1 libindicator7
RUN wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | sudo apt-key add -
RUN echo "deb http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google-chrome.list
#
# Expected version is 41.0.2272.76-1, but specifying on the command line doesn't always work :-(
RUN apt-get update -qq && apt-get install -y -q --force-yes google-chrome-stable

#
# Setup Xvfb - FF and chrome will connect to this DISPLAY
# (https://github.com/keyvanfatehi/docker-chrome-xvfb)
#
ENV DISPLAY :99
ADD xvfb_init /etc/init.d/xvfb
RUN chmod a+x /etc/init.d/xvfb

#
# Add the script that sets up a user account to run cucumber. The account has to be setup
# on each execution so that the uid/gid can be synced with the caller. Otherwise, the files
# created by cucumber/capybara will not be owned by the right user.
#
ADD makeCukeUser.sh /usr/sbin/makeCukeUser.sh
RUN chmod a+x /usr/sbin/makeCukeUser.sh

#
# Add the script used to run cucumber.
#
ADD runCucumber.sh /usr/sbin/runCucumber.sh
RUN chmod a+x /usr/sbin/runCucumber.sh

#
# Add the Java app that generates a nice HTML report
#
RUN apt-get update -qq && apt-get install -y -q openjdk-7-jre-headless
RUN mkdir -p /usr/share/reporter
ADD reporter.jar /usr/share/reporter/reporter.jar

#
# This is the directory that wlll be the mount point for the cucumber files
#
WORKDIR /capybara
