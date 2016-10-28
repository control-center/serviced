Then (/^VHost "([^"]*)" should exist$/) do |vhost|
    expect(CC.UI.ServicesPage.check_vhost_exists?(vhost))
end

Then (/^Public Port "([^"]*)" should exist$/) do |port|
    expect(CC.UI.ServicesPage.check_public_port_exists?(port))
end

Then (/^I should see the Add Public Endpoint dialog$/) do
    expect(CC.UI.ServicesPage.addPublicEndpoint_dialog.visible?).to be true
end

Then (/^I choose Port type$/) do
    CC.UI.ServicesPage.buttonPortType.click
end

Then (/^I choose VHost type$/) do
    CC.UI.ServicesPage.buttonVHostType.click
end

Then (/^I select Service "([^"]*)" - "([^"]*)"$/) do |service, endpoint|
    val = getTableValue(service)  + " - " + getTableValue(endpoint)
    CC.UI.ServicesPage.addVHostApp_select.select val
end

Then (/^I fill in Host "([^"]*)"$/) do |host|
    host=getTableValue(host)
    CC.UI.ServicesPage.newHostName_input.set host
    expect(CC.UI.ServicesPage.newHostName_input.value.should eq host)
end

Then (/^I fill in Port "([^"]*)"$/) do |port|
    port = getTableValue(port)
    CC.UI.ServicesPage.newPort_input.set port
    expect(CC.UI.ServicesPage.newPort_input.value.should eq port)
end

Then (/^I should see Port "([^"]*)" to be "([^"]*)"$/) do |attr, val|
    expect(CC.UI.ServicesPage.newPort_input[attr].eql?(getTableValue(val)))
end

Then (/^I should see Host "([^"]*)" to be "([^"]*)"$/) do |attr, val|
    expect(CC.UI.ServicesPage.newHostName_input[attr].eql?(getTableValue(val)))
end

Then (/^I select Protocol "([^"]*)"$/) do |protocol|
    CC.UI.ServicesPage.addProtocol_select.select getTableValue(protocol)
end

Then (/^I should see Port fields$/) do
    expect(CC.UI.ServicesPage.addVHostApp_select.visible?).to be true
    expect(CC.UI.ServicesPage.newHostName_input.visible?).to be true
    expect(CC.UI.ServicesPage.newPort_input.visible?).to be true
    expect(CC.UI.ServicesPage.addProtocol_select.visible?).to be true
    expect(CC.UI.ServicesPage.buttonPortType.visible?).to be true
    expect(CC.UI.ServicesPage.buttonVHostType.visible?).to be true
end

Then (/^I should see VHost fields$/) do
    expect(CC.UI.ServicesPage.addVHostApp_select.visible? && CC.UI.ServicesPage.newVHost_input.visible?).to be true
end

Then (/^"([^"]*)" should be selected for Service Endpoint$/) do |expected|
    expect(CC.UI.ServicesPage.addVHostApp_select.find('option[selected]')).to have_content getTableValue(expected)
end

Then (/^"([^"]*)" should be selected for Protocol$/) do |expected|
    expect(CC.UI.ServicesPage.addProtocol_select.find('option[selected]')).to have_content getTableValue(expected)
end

Then (/^I should see only one endpoint entry of Port "([^"]*)"$/) do |port|
    port = getTableValue(port)
    expect(CC.UI.ServicesPage.check_endpoint_unique_column?('URL', port))
end

Then (/^I should see the endpoint entry with both "([^"]*)" and "([^"]*)"$/) do |c1, c2|
    c1 = getTableValue(c1)
    c2 = getTableValue(c2)
    expect(CC.UI.ServicesPage.check_endpoint_find?(c1, c2))
end

Then (/^I delete Endpoint "([^"]*)"$/) do |entry|
    expect(CC.UI.ServicesPage.remove_publicendpoint?(entry)).to be(true)
end

When (/^I click the Add Public Endpoint button$/) do
    CC.UI.ServicesPage.click_add_publicendpoint_button()
end
