require_relative 'navbar'
require 'site_prism'

class Service < SitePrism::Page
    include ::RSpec::Matchers

    set_url applicationURL("#/services?disable-animation=true&loglevel=debug")
    set_url_matcher /services/

    section :navbar, NavBarSection, ".navbar-collapse"

    element :addPublicEndpoint_button, "[ng-click='modalAddPublicEndpoint()']"
    element :publicEndpoints_table, "table[data-config='publicEndpointsTable']"
    element :ipAssignments_table,   "table[data-config='ipsTable']"
    element :configFiles_table,     "table[data-config='configTable']"

    # add public endpoint dialog elements
    element :addPublicEndpoint_dialog, ".modal-content"
    element :addVHostApp_select,    :xpath, "//select[@id='add_vhost_application']"

    # type selector
    element :buttonPortType,        "div.btn-group", "//input[@value='port']"
    element :buttonVHostType,       "div.btn-group", "//input[@value='vhost']"

    # type port
    element :newHostName_input,     :xpath, "//input[@name='new_host_name']"
    element :newPort_input,         :xpath, "//input[@id='add_public_endpoint_port']"
    element :addProtocol_select,    :xpath, "//select[@id='add_port_protocol']"

    # type vhost
    element :newVHost_input,        :xpath, "//input[@id='add_vhost_vhost']"

    # Look up the table data for the given port and remove it using
    # the UI.
    def remove_publicendpoint_port_json(name)
        expect(remove_publicendpoint?("table://ports/#{name}/PortAddr")).to be(true)
    end

    # Look up the table data for the given vhost and remove it using
    # the UI.
    def remove_publicendpoint_vhost_json(name)
        expect(remove_publicendpoint?("table://vhosts/#{name}/Name")).to be(true)
    end

    # Removes the public endpoint (vhost or port) by looking up the entry and
    # clicking the delete button.
    def remove_publicendpoint?(name)
        name = getTableValue(name)
        self.page.all(:xpath, "//table[@data-config='publicEndpointsTable']//tr").each do |tr|
            if tr.text.include?(name)
                btn = tr.find(:xpath, ".//button[@ng-click='clickRemovePublicEndpoint(publicEndpoint)']")
                if btn
                    btn.click
                    # confirm the removal
                    cnf = find(:xpath, "//div[@class='modal-content']//button", :text => "Remove")
                    cnf.click
                    refreshPage()
                    return true
                end
            end
        end
        return false
    end

    def click_add_publicendpoint_button()
        self.addPublicEndpoint_button.click
        # wait till modal is done loading
        expect(self).to have_no_css(".uilock", :visible => true)
    end

    def check_endpoint_unique_column?(ctitle, cvalue)
        found = 0
        self.page.all(:xpath, "//table[@data-config='publicEndpointsTable']//tr//td[@data-title-text=#{ctitle}]").each do |td|
            if td.text.include?(cvalue)
                found += 1
            end
        end
        return found == 1
    end

    def check_endpoint_find?(c1, c2)
        delay = 3   # seconds
        maxRetries = 2
        begin
            retries ||= 0
            self.page.all(:xpath, "//table[@data-config='publicEndpointsTable']//tr").each do |tr|
                line=tr.text.upcase()
                if  line.include?(c1) && line.include?(c2)
                    return true
                end
            end

        # For whatever reason, this method sometimes fails with a 'stale element reference' error, which can mean
        # that in between the time we find an element and try to reference, it's been deleted.
        # So we use a retry here as a work around.
        # For more information, google "selenium stale element reference".
        rescue Selenium::WebDriver::Error::StaleElementReferenceError
            sleep delay
            retry if (retries += 1) < maxRetries
        end
        return false
    end


    def check_vhost_exists?(vhost)
        vhostName = getTableValue(vhost)
        searchStr = "https://#{vhostName}."

        found = false
        within(self.publicEndpoints_table) do
            found = page.has_text?(searchStr)
        end
        return found
    end

    def check_public_port_exists?(port)
        portName = getTableValue(port)
        searchStr = ":#{portName}"

        found = false
        within(self.publicEndpoints_table) do
            found = page.has_text?(searchStr)
        end
        return found
    end
end
