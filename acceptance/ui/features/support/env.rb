#
# This file is responsible for loading/executing any code that exists outside of
# the step definitions. You can also define hooks either here or in another file
# in this directory (e.g. hooks.rb). For more info on hooks, see
# https://github.com/cucumber/cucumber/wiki/Hooks
#
require 'fileutils'
require "capybara"
require "capybara/cucumber"
require "capybara/rspec"
require 'capybara/poltergeist'
require 'selenium-webdriver'
require 'capybara-screenshot/cucumber'
require 'capybara-webkit'

def combineAllDataFiles(dir)
    data = "{"
    Dir.foreach(dir) do |file|
        next if file == '.' || file == '..'
        begin
            original = File.read(File.join(dir, file))
        rescue => err
            printf "ERROR: Dataset file %s could not be read: %s\n", file, err.message
            exit 1
        end
        data << removeWhitespaceAndOuterBrackets(original)
        data << ",\n\n"
    end
    data = data.rstrip.chop
    data << "\n}"
    return data
end

def removeWhitespaceAndOuterBrackets(text)
    text = text.strip
    text = (text[1..-2]).rstrip
    return text
end

def parseJson(data)
    begin
        data = JSON.parse(data)
    rescue => err
        printf "ERROR: Dataset file could not be parsed: %s\n", err.message
        exit 1
    end
    return data
end

#
# Set defaults
Capybara.default_max_wait_time = 10
Capybara.default_driver = :selenium
Capybara::Screenshot.prune_strategy = :keep_last_run

Capybara.app_host = ENV["APPLICATION_URL"]
if Capybara.app_host.empty?
    #
    # replace with our own default application as necessary
    Capybara.app_host = "http://localhost"
end
printf "Using app_host=%s\n", Capybara.app_host
printf "Using userid=%s\n", ENV["APPLICATION_USERID"]

#
# NOTE: Cucumber does not provide an API to override the output directory
#       for the various foramatters, so we don't use the OUTPUT_DIR env var
#       as an override like the other environment variables. Instead, we
#       use it to set Capybara.save_and_open_page_path assuming that the value
#       of OUTPUT_DIR and the value specified for the --out argument for the
#       HTML formatter are the same.
#
output_dir = ENV["OUTPUT_DIR"]
if output_dir.nil? || output_dir.empty?
    printf "ERROR: OUTPUT_DIR is not defined; check cucumber.yml"
    exit 1
end

if Dir.exists?(output_dir)
    FileUtils.rm_rf(output_dir)
end
Capybara.save_and_open_page_path = output_dir + "/screenshots"
FileUtils.mkdir_p(Capybara.save_and_open_page_path)
printf "Using output directory=%s\n", output_dir

dataset_dir = File.join(ENV["DATASET_DIR"], ENV["DATASET"])
if !Dir.exists?(dataset_dir) || Dir.entries(dataset_dir).size <= 2
    printf "ERROR: DATASET_DIR is not defined; check cucumber.yml\n"
    exit 1
end

data = combineAllDataFiles(dataset_dir)
PARSED_DATA = parseJson(data)

template_dir = File.join(ENV["DATASET_DIR"], "/testsvc")
if !Dir.exists?(template_dir) || Dir.entries(template_dir).size <= 2
    printf "ERROR: #{template_dir} is not defined\n"
    exit 1
end
TEMPLATE_DIR = template_dir

printf "Using dataset directory=%s\n", ENV["DATASET_DIR"]
printf "Using dataset=%s\n", ENV["DATASET"]
printf "Using template directory=%s\n", template_dir

timeout_override = ENV["CAPYBARA_TIMEOUT"]
if timeout_override && timeout_override.length > 0
    Capybara.default_max_wait_time = timeout_override.to_i
end
printf "Using default_max_wait_time=%d\n", Capybara.default_max_wait_time

driver_override = ENV["CAPYBARA_DRIVER"]
if driver_override && driver_override.length > 0
    if driver_override == "selenium"
        Capybara.default_driver = :selenium
    elsif driver_override == "selenium_chrome"
        Capybara.default_driver = :selenium_chrome
    elsif driver_override == "poltergeist"
        Capybara.default_driver = :poltergeist
    elsif driver_override == "webkit"
        Capybara.default_driver = :webkit
        Capybara.javascript_driver = :webkit
    else
        puts "ERROR: invalid value for CAPYBARA_DRIVER"
        exit 1
    end
end
printf "Using driver=%s\n", Capybara.default_driver

HOST_IP = ENV["HOST_IP"]
TARGET_HOST = ENV["TARGET_HOST"]
printf "Using HOST_IP=%s\n", HOST_IP
printf "Using TARGET_HOST=%s\n", TARGET_HOST
printf "Using DISPLAY=%s\n", ENV["DISPLAY"]

#
# Register Chrome (Firefox is the selenium default)
Capybara.register_driver :selenium_chrome do |app|
    # Get the default capabilities which enable javascript and browser logging
    # For whatever reasaon, we have to explicitly specify these capabilities to enable browser logging
    caps = Selenium::WebDriver::Remote::Capabilities.chrome()

    #
    # Chrome's sandboxing doesn't work in a docker container because Chrome detects it's on an SID volume
    # and refuses to start. So you must either run the docker container w/--privileged or run chrome as
    # a non-root user with sandboxing disabled. The later seems most secure
    Capybara::Selenium::Driver.new(app,
                                   :browser => :chrome,
                                   :args => ["--no-sandbox", "--ignore-certificate-errors", "--user-data-dir=/tmp"],
                                   :desired_capabilities => caps
    )
end

#
# Register poltergeist (headless driver based on phantomjs)
Capybara.register_driver :poltergeist do |app|
    options = {
        :ignore_ssl_errors => true,
        :js_errors => true,
        :debug => false,
        :timeout => 120,
        :phantomjs_options => ['--load-images=no', '--disk-cache=false', '--ignore-ssl-errors=true'],
        # FIXME: Write a custom logger that implements the Ruby IO object so that we can cache all of the messages and
        #        produce scenario-specific reports ala the selenium-chrome driver (see hooks.rb)
        # Uncomment to get JS console messages written to stdout
        # :phantomjs_logger => STDOUT,
        :inspector => true,
    }
    Capybara::Poltergeist::Driver.new(app, options)
end

Capybara::Webkit.configure do |config|
    # Allow loading of all external URLs
    config.allow_url("*")
    config.ignore_ssl_errors = true
end

Before do
  # Any DISPLAY value other than ":99" implies we're xhosting the browser outside of the docker container
  debug = false
  if ENV["DISPLAY"] != ":99"
        debug = true
  end

  if !debug && (Capybara.current_driver == :selenium || Capybara.current_driver == :selenium_chrome)
    require 'headless'

    headless = Headless.new
    headless.start
  end
end

#
# Required so that Screenshot works with the "selenium_chrome" driver
Capybara::Screenshot.register_driver(:selenium_chrome) do |driver, path|
    driver.browser.save_screenshot(path)
end

# Turns off default screenshots (taken after cleanup hooks)
# so only one screenshot per failure is taken
Capybara::Screenshot.autosave_on_failure = false
