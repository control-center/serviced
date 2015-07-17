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
Capybara.default_wait_time = 10
Capybara.default_driver = :selenium
Capybara::Screenshot.prune_strategy = :keep_last_run

Capybara.app_host = ENV["APPLICATION_URL"]
if Capybara.app_host.empty?
    #
    # replace with our own default application as necessary
    Capybara.app_host = "http://localhost"
end
printf "Using app_host=%s\n", Capybara.app_host

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

printf "Using dataset directory=%s\n", ENV["DATASET_DIR"]
printf "Using dataset=%s\n", ENV["DATASET"]

timeout_override = ENV["CAPYBARA_TIMEOUT"]
if timeout_override && timeout_override.length > 0
    Capybara.default_wait_time = timeout_override.to_i
end
printf "Using default_wait_time=%d\n", Capybara.default_wait_time

driver_override = ENV["CAPYBARA_DRIVER"]
if driver_override && driver_override.length > 0
    if driver_override == "selenium"
        Capybara.default_driver = :selenium
    elsif driver_override == "selenium_chrome"
        Capybara.default_driver = :selenium_chrome
    elsif driver_override == "poltergeist"
        Capybara.default_driver = :poltergeist
    else
        puts "ERROR: invalid value for CAPYBARA_DRIVER"
        exit 1
    end
end
printf "Using driver=%s\n", Capybara.default_driver

HOST_IP = ENV["HOST_IP"]
printf "Using IP Address=%s\n", HOST_IP

#
# Register Chrome (Firefox is the selenium default)
Capybara.register_driver :selenium_chrome do |app|
    #
    # Chrome's sandboxing doesn't work in a docker container because Chrome detects it's on an SID volume
    # and refuses to start. So you must either run the docker container w/--privileged or run chrome as
    # a non-root user with sandboxing disabled. The later seems most secure
    args = %w(--no-sandbox)
    Capybara::Selenium::Driver.new(app, :browser => :chrome, :args => ["--no-sandbox", "--ignore-certificate-errors"])
end

#
# Register poltergeist (headless driver based on phantomjs)
Capybara.register_driver :poltergeist do |app|
    options = {
    	:ignore_ssl_errors => true,
        :js_errors => false,
        :timeout => 120,
        :debug => false,
        :phantomjs_options => ['--load-images=no', '--disk-cache=false', '--ignore-ssl-errors=true'],
        :inspector => true,
    }
    Capybara::Poltergeist::Driver.new(app, options)
end

#
# Required so that Screenshot works with the "selenium_chrome" driver
Capybara::Screenshot.register_driver(:selenium_chrome) do |driver, path|
    driver.browser.save_screenshot(path)
end
