
#
# Modifications to the selenium behavior for our tests.
#

class Capybara::Selenium::Driver
    # The selenium driver doesn't unload itself, it just clears the cookies to "reset"
    # the session.  If we override this method, we can prevent that and use the same login
    # for all tests.
    # ref: https://github.com/jnicklas/capybara/blob/master/lib/capybara/selenium/driver.rb#L93
    def reset!
        # Use instance variable directly so we avoid starting the browser just to navigate to about:blank
        if @browser
            navigated = false
            if !navigated
                # Don't reset.. but do navigate to a blank page.
                @browser.navigate.to("about:blank")
                navigated = true
            end
        end
    end
end
