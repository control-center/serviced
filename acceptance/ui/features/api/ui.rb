require "capybara"
require "capybara/rspec"
require_relative "../pages/applications"
require_relative "../pages/hosts"
require_relative "../pages/pools"
require_relative "../pages/service"
require_relative "../pages/user"

#
# Contains the list of page objects to be returned.  Login is only
# done if the session is invalid.
#
class UI
    include ::RSpec::Matchers
    include ::Capybara::DSL

    def initialize()
        @pages = {
            applications: Applications.new,
            hosts: Hosts.new,
            login: nil,
            pools: Pools.new,
            services: Service.new,
            user: User.new,
        }
    end

    # Ensures that we've logged in.  Subsequent calls
    # will use the same page and not login again.
    def login()
        if self.LoginPage == nil || !verify_login?()
            puts "  \e[33m** Logging in to the UI **\e[0m"
            login_as(applicationUserID(), applicationPassword())
        end
    end

    def login_as(username, password)
        @pages[:login] = nil # Reset the login data.
        do_login(username, password)
    end

    def visit_login_page()
        @pages[:login] = Login.new if self.LoginPage == nil
        visit_login_page_impl(self.LoginPage)
    end

    def LoginPage
        return @pages[:login]
    end

    def HostsPage
        suppress_deploy_wizard()
        return @pages[:hosts]
    end

    def ApplicationsPage
        suppress_deploy_wizard()
        return @pages[:applications]
    end

    def PoolsPage
        suppress_deploy_wizard()
        return @pages[:pools]
    end

    def ServicesPage
        suppress_deploy_wizard()
        return @pages[:services]
    end

    def UserPage
        suppress_deploy_wizard()
        return @pages[:user]
    end

    private

    ##
    # Login methods.
    #

    # Tries to access the pools page.  If we're redirected to the
    # login page, we need to login.
    def verify_login?()
        @pages[:pools].load # can't access the  property here since that tries to set cookies.
        return URI.parse(current_url).to_s.index("login") == nil
    end

    def do_login(username, password)
        login = Login.new

        visit_login_page_impl(login)
        suppress_deploy_wizard()
        fill_in_default_user_id(login, username)
        fill_in_default_password(login, password)
        click_signin_button(login)

        # Finally set the login variable
        @pages[:login] = login
    end

    def visit_login_page_impl(login)
        oldWait = setDefaultWaitTime(180)

        login.load
        expect(login).to be_displayed

        setDefaultWaitTime(oldWait)

        # wait till loading animation clears
        login.has_no_css?(".loading_wrapper")
    end

    def fill_in_default_user_id(login, username)
        login.userid_field.set username
    end

    def fill_in_default_password(login, password)
        login.password_field.set password
    end

    def click_signin_button(login)
        login.signin_button.click
    end

    def dump_cookies()
        if page.driver.browser.manage.all_cookies != nil
            page.driver.browser.manage.all_cookies.each do |cookie|
                printf "....cookie: %s\n",cookie[:name]
            end
            return
        else
            printf "....no cookies\n"
        end
    end

    def suppress_deploy_wizard()
        # dump_cookies()
        deploywizcookie = page.driver.browser.manage.cookie_named("autoRunWizardHasRun")
        if deploywizcookie != nil
            return
        end
        page.driver.browser.manage.add_cookie(
            {name:"autoRunWizardHasRun", value:"true"}
        )
    end
end
