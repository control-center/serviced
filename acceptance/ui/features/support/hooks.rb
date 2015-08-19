require "capybara-screenshot"
#
# See https://github.com/cucumber/cucumber/wiki/Hooks for more info about hooks
#
Before('@login-required') do
    visitLoginPage()
    fillInDefaultUserID()
    fillInDefaultPassword()
    clickSignInButton()
end

After('@clean_hosts') do
    removeAllHostsCLI()
end

After('@clean_pools') do
    visitPoolsPage()
    removeAllPools()
    addDefaultPool() # default pool must exist or else serviced log gets spammed CC-1105
end

After('@clean_templates') do
    visitApplicationsPage()
end

After('@clean_services') do
    visitApplicationsPage()
    removeAllEntries("service")
end

After('@clean_virtualips') do
    if (@pools_page.virtualIps_table.has_no_text?("No Data Found"))
        removeAllEntries("address")
    end
end

After('@screenshot') do |scenario|
  if scenario.failed?
    Capybara.using_session(Capybara::Screenshot.final_session_name) do
      filename_prefix = Capybara::Screenshot.filename_prefix_for(:cucumber, scenario)

      saver = Capybara::Screenshot::Saver.new(Capybara, Capybara.page, true, filename_prefix)
      saver.save
      saver.output_screenshot_path

      if File.exist?(saver.screenshot_path)
        require "base64"
        image = open(saver.screenshot_path, 'rb') {|io|io.read}
        encoded_img = Base64.encode64(image)
        embed(encoded_img, 'image/png;base64', "Screenshot of the error")
      end
    end
  end
end
