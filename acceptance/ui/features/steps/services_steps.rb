Then (/^VHost "(.*?)" should exist$/) do |vhost|
    expect(checkVHostExists(vhost))
end

Then (/^Public Port "(.*?)" should exist$/) do |port|
    expect(checkPublicPortExists(port))
end

Then (/^I should see the Add Public Endpoint dialog$/) do
    if @service_page == nil
         @service_page = Service.new
    end
    @service_page.addPublicEndpoint_dialog.visible?
end

Then (/^I choose Port type$/) do
    @service_page.find("div.btn-group", "//input[@value='port']" ).click
end

Then (/^I choose VHost type$/) do
    @service_page.find("div.btn-group", "//input[@value='vhost']" ).click
end

Then (/^I should find all fields$/) do
    @service_page.has_addVHostApp_select?   &&
    @service_page.has_newHostName_input?    &&
    @service_page.has_newPort_input?        &&
    @service_page.has_addProtocol_select?   &&
    @service_page.has_addVHostApp_select?   &&
    @service_page.has_newVHost_input?
end

Then (/^I should see Port fields$/) do
    @service_page.addVHostApp_select.visible? &&
    @service_page.newHostName_input.visible? &&
    @service_page.newPort_input.visible? &&
    @service_page.addProtocol_select.visible?
end

Then (/^I should see VHost fields$/) do
    @service_page.addVHostApp_select.visible? &&
    @service_page.newVHost_input.visible?
end

def checkVHostExists(vhost)
    initServicePageObj()
    vhostName = getTableValue(vhost)
    searchStr = "https://#{vhostName}."

    found = false
    within(@service_page.publicEndpoints_table) do
        found = page.has_text?(searchStr)
    end
    return found
end

def checkPublicPortExists(port)
    initServicePageObj()
    portName = getTableValue(port)
    searchStr = ":#{portName}"

    found = false
    within(@service_page.publicEndpoints_table) do
        found = page.has_text?(searchStr)
    end
    return found
end

def initServicePageObj()
    if @service_page.nil?
        @service_page = Service.new
    end
end
