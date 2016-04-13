require "capybara-screenshot"

# Write a message to the JS console so that we know when a scenario starts
Before do |scenario|

  window = Capybara.page.driver.browser.manage.window
  if window != nil
    window.resize_to(1280, 1024)
  end

  cmd = sprintf "console.log(\"%s\")", getStartScenarioAnnouncement(scenario)
  Capybara.page.driver.execute_script(cmd)
end

#
# See https://github.com/cucumber/cucumber/wiki/Hooks for more info about hooks
#
Before('@login-required') do
  loginAsDefaultUser()
end

After('@clean_hosts') do
  removeAllHostsCLI()
end

After('@clean_pools') do
  removeAllPoolsExceptDefault()
end

After('@clean_templates') do
  removeAllTemplatesCLI()
end

After('@clean_services') do
  removeAllServicesCLI()
end

After('@clean_virtualips') do
  removeVirtualIPsFromDefaultPoolCLI()
end

After('@screenshot') do |scenario|
  if scenario.failed?
    Capybara.using_session(Capybara::Screenshot.final_session_name) do
      printf "Saving screenshot for scenario '%s' ...\n", scenario.name
      filename_prefix = Capybara::Screenshot.filename_prefix_for(:cucumber, scenario.name)

      saver = Capybara::Screenshot::Saver.new(Capybara, Capybara.page, true, filename_prefix)
      saver.save
      saver.output_screenshot_path

      if File.exist?(saver.screenshot_path)
        require "base64"
        image = open(saver.screenshot_path, 'rb') {|io|io.read}
        encoded_img = Base64.encode64(image)
        embed(encoded_img, 'image/png;base64', "Screenshot of the error")
      end

      #
      # FIXME: The syntax for accessing console logs varies by driver. Need to add support for other
      #        drivers as needed.
      # Notes:
      # 1. Using this technique for the FF selenium driver captures lots of log messages, but the none of them
      #    are from the JS console. Apparently, they are from the driver and/or FF itself. Capturing JS console
      #    from FF seems to require some additional FF plugins
      # 2. Poltergeist logs to stdout. If we installed our own logger which cached a list of messages kind of like
      #    the chromedriver does. then we could implement a variation of the logic below for poltergeist as well.
      if Capybara.current_driver == :selenium_chrome
        log_path = getLogFilePath(scenario.name)
        embedLinkToConsoleLog(log_path)
        log_file = File.open(log_path, "w")

        printf "\n=======================================================================================\n"
        printf "Here is the output from the browser console:\n"
        printf "--------------------------------------------\n"
        doReport = false
        startScenario = getStartScenarioAnnouncement(scenario)
        endScenario = getFinishScenarioAnnouncement(scenario)

        # log all of the JS console messages for the current scenario
        log_entries = Capybara.page.driver.browser.manage.logs.get(:browser)
        log_entries.each do |entry|
          if entry.message.include? startScenario
            doReport = true
          end

          if doReport == true
            printf "%s\n", entry.to_json
            log_file.printf "%s\n", entry.to_json
          end

          if entry.message.include? endScenario
            doReport = false
          end
        end
        printf "=======================================================================================\n"

        log_file.close
      end
    end
  end
end

# Write a message to the JS console so that we know when a scenario finishes
After do |scenario|
  cmd = sprintf "console.log(\"%s\")", getFinishScenarioAnnouncement(scenario)
  Capybara.page.driver.execute_script(cmd)
end

def getStartScenarioAnnouncement(scenario)
  return sprintf "Starting Cucumber Scenario '%s'", scenario.name
end

def getFinishScenarioAnnouncement(scenario)
  return sprintf "Finished Cucumber Scenario '%s'", scenario.name
end

# Get the path to the file used to store JS log messages
def getLogFilePath(scenario_name)
  # Use a directory named 'logs' which is a sibling of the 'screenshots' directory
  log_directory = Capybara.save_and_open_page_path.sub("screenshots", "logs")
  if !Dir.exists?(log_directory)
    FileUtils.mkdir_p(log_directory)
  end

  # make a 'safe' file name by replacing all non-ascii, non-alphanumeric with underscore
  safe_scenario_name = scenario_name.gsub(/[^0-9A-Za-z.\-]/, '_')
  log_file_name = sprintf "%s.log", safe_scenario_name
  log_path = File.join(log_directory, log_file_name)
  return log_path
end

# Update the Cucumber report to include a link to the file containing the console log messages
def embedLinkToConsoleLog(log_path)
  link_path = log_path.sub('output/', '')
  log_link = sprintf("<a href='%s'>%s</a>", link_path, log_path)
  embed(log_link, 'text/plain', "JS console messages")
end
