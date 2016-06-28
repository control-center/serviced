# Acceptance tests to verify CLI command behavior.

Given (/^(?:|that )the "(.*)" port is added$/) do |name|
    CC.CLI.service.add_publicendpoint_port_json(name) if !CC.CLI.service.check_publicendpoint_port_exists_json(name)
end

Given (/^(?:|that )the "([^"]*)" port does not exist$/) do |name|
    CC.CLI.service.remove_publicendpoint_port_json(name) if CC.CLI.service.check_publicendpoint_port_exists_json(name)
end

Given (/^(?:|that )the port public endpoint "([^"]*)" is removed$/) do |name|
    CC.CLI.service.remove_publicendpoint_port_json(name)
end

Given (/^(?:|that )the (port|vhost) public endpoint "([^"]*)" is (enabled|disabled)$/) do |pepType, name, enabled|
    enabled = (enabled == "enabled" ? true : false)
    CC.CLI.service.enable_publicendpoint_port_json(name, enabled) if pepType == "port"
    CC.CLI.service.enable_publicendpoint_vhost_json(name, enabled) if pepType == "vhost"
end

Given (/^(?:|that )the "(.*)" vhost is added$/) do |name|
    CC.CLI.service.add_publicendpoint_vhost_json(name) if !CC.CLI.service.check_publicendpoint_vhost_exists_json(name)
end

Given (/^(?:|that )the "([^"]*)" vhost does not exist$/) do |name|
    CC.CLI.service.remove_publicendpoint_vhost_json(name) if CC.CLI.service.check_publicendpoint_vhost_exists_json(name)
end

Given (/^(?:|that )the vhost public endpoint "([^"]*)" is removed$/) do |name|
    CC.CLI.service.remove_publicendpoint_vhost_json(name)
end

Then(/^I should see the (port|vhost) public endpoint (.*) in the list output$/) do |pepType, name|
    CC.CLI.service.verify_publicendpoint_port_list_matches(name) if pepType == "port"
    CC.CLI.service.verify_publicendpoint_vhost_list_matches(name) if pepType == "vhost"
end

Then(/^I should see the (port|vhost) public endpoint "([^"]*)" in the service$/) do |pepType, name|
    expect(CC.CLI.service.check_publicendpoint_port_exists_json(name)).to_not be(nil) if pepType == "port"
    expect(CC.CLI.service.check_publicendpoint_vhost_exists_json(name)).to_not be(nil) if pepType == "vhost"
end

Then(/^I should not see the (port|vhost) public endpoint "([^"]*)" in the service$/) do |pepType, name|
    expect(CC.CLI.service.check_publicendpoint_port_exists_json(name)).to be(nil) if pepType == "port"
    expect(CC.CLI.service.check_publicendpoint_vhost_exists_json(name)).to be(nil) if pepType == "vhost"
end

Then(/^the (port|vhost) public endpoint "([^"]*)" should be "([^"]*)" in the service$/) do |pepType, name, enabled|
    enabled = (enabled == "enabled" ? true : false)
    expect(CC.CLI.service.check_publicendpoint_port_enabled_in_service_json?(name)).to be(enabled) if pepType == "port"
    expect(CC.CLI.service.check_publicendpoint_vhost_enabled_in_service_json?(name)).to be(enabled) if pepType == "vhost"
end
