Then (/^VHost "([^"]*)" should exist$/) do |vhost|
    expect(checkVHostExists(vhost))
end

Then (/^Public Port "([^"]*)" should exist$/) do |port|
    expect(checkPublicPortExists(port))
end

Then (/^I should see the Add Public Endpoint dialog$/) do
    initServicePageObj()
    expect(@service_page.addPublicEndpoint_dialog.visible?).to be true
end

Then (/^I choose Port type$/) do
    @service_page.buttonPortType.click
end

Then (/^I choose VHost type$/) do
    @service_page.buttonVHostType.click
end

Then (/^I select Service "([^"]*)" - "([^"]*)"$/) do |service, endpoint|
    val = getTableValue(service)  + " - " + getTableValue(endpoint)
    @service_page.addVHostApp_select.select val
end

Then (/^I fill in Host "([^"]*)"$/) do |host|
    @service_page.newHostName_input.set getTableValue(host)
end

Then (/^I fill in Port "([^"]*)"$/) do |port|
    @service_page.newPort_input.set getTableValue(port)
end

Then (/^I should see Port "([^"]*)" to be "([^"]*)"$/) do |attr, val|
    expect(@service_page.newPort_input[attr].eql?(getTableValue(val)))
end

Then (/^I should see Host "([^"]*)" to be "([^"]*)"$/) do |attr, val|
    expect(@service_page.newHostName_input[attr].eql?(getTableValue(val)))
end

Then (/^I select Protocol "([^"]*)"$/) do |protocol|
    @service_page.addProtocol_select.select getTableValue(protocol)
end

Then (/^I should see Port fields$/) do
    initServicePageObj()

    expect(@service_page.addVHostApp_select.visible?).to be true
    expect(@service_page.newHostName_input.visible?).to be true
    expect(@service_page.newPort_input.visible?).to be true
    expect(@service_page.addProtocol_select.visible?).to be true
    expect(@service_page.buttonPortType.visible?).to be true
    expect(@service_page.buttonVHostType.visible?).to be true

end

Then (/^I should see VHost fields$/) do
    initServicePageObj()
    expect(@service_page.addVHostApp_select.visible? && @service_page.newVHost_input.visible?).to be true
end

Then (/^"([^"]*)" should be selected for Service Endpoint$/) do |expected|
    initServicePageObj()
    expect(@service_page.addVHostApp_select.find('option[selected]')).to have_content getTableValue(expected)
end

Then (/^"([^"]*)" should be selected for Protocol$/) do |expected|
    initServicePageObj()
    expect(@service_page.addProtocol_select.find('option[selected]')).to have_content getTableValue(expected)
end

Then (/^Endpoint details should be service:"([^"]*)" endpoint:"([^"]*)" type:"([^"]*)" protocol:"([^"]*)" host:"([^"]*)" port:"([^"]*)"$/) do |svc, ep, type, proto, host, port|
    svc=getTableValue(svc).upcase()
    ep=getTableValue(ep).upcase()
    type=getTableValue(type).upcase()
    proto=getTableValue(proto).upcase()
    host=getTableValue(host).upcase()
    port=getTableValue(port).upcase()
    expect(checkEndpointRow(svc, ep, type, proto, host, port)).to be true
end

Then (/^I delete Endpoint "([^"]*)"$/) do |entry|
    deleteEndpoint(entry)
end

def deleteEndpoint(name)
    initServicePageObj()
    name = getTableValue(name)
    @service_page.all(:xpath, "//table[@data-config='publicEndpointsTable']//tr").each do |tr|
        if tr.text.include?(name)
            btn = tr.find(:xpath, ".//button[@ng-click='clickRemovePublicEndpoint(publicEndpoint)']")
            if btn
                btn.click
                # confirm the removal
                cnf = find(:xpath, "//div[@class='modal-content']//button", :text => "Remove and Restart Service")
                cnf.click
                refreshPage()
                return true
            end
        end
    end
    return false
end

def checkEndpointRow(svc, ep, type, proto, host, port)
    @service_page.all(:xpath, "//table[@data-config='publicEndpointsTable']//tr").each do |tr|
        line=tr.text.upcase()
        if  line.include?(svc) && line.include?(ep) && line.include?(type) && line.include?(proto) && line.include?(port)
            return true
        end
    end
    return false
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
