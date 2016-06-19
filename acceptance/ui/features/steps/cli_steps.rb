# Acceptance tests to verify CLI command behavior.

Given (/^(?:|that )the "(.*)" port is added$/) do |name|
    CC.CLI.add_publicendpoint_port_json(name) if !CC.CLI.check_publicendpoint_port_exists_json(name)
end

Given (/^(?:|that )the "([^"]*)" port does not exist$/) do |name|
    # Once the remove CLI method is ready the CLI can be used for this Given condition.
    #CC.CLI.remove_publicendpoint_port_json(name) if CC.CLI.check_publicendpoint_port_exists_json(name)

    CC.UI.login()
    CC.UI.remove_publicendpoint_port_json(name) if CC.CLI.check_publicendpoint_port_exists_json(name)
end

Given (/^(?:|that )the port public endpoint "([^"]*)" is removed$/) do |name|
    # Once the remove CLI method is ready the CLI can be used for this Given condition.
    #CC.CLI.remove_publicendpoint_port_json(name)

    CC.UI.login()
    CC.UI.remove_publicendpoint_port_json(name)
end

When(/^I should see the (.*) public endpoint (.*) in the service$/) do |pepType, name|
    CC.CLI.verify_publicendpoint_port_list_matches(name) if pepType == "port"
    #CC.CLI.verify_publicendpoint_vhost_list_matches(name) if pepType == "vhost"
end

Then(/^I should not see the (.*) public endpoint "([^"]*)" in the service$/) do |pepType, name|
    expect(CC.CLI.check_publicendpoint_port_exists_json(name)).to be(nil) if pepType == "port"
    #expect(CC.CLI.check_publicendpoint_vhost_exists_json(name)).to be(nil) if pepType == "vhost"
end
