Given (/^(?:|that )multiple hosts have been added$/) do
    visitHostsPage()
    CC.UI.HostsPage.wait_for_host_entries(getDefaultWaitTime())
    if CC.UI.HostsPage.host_entries.size < 5
        CC.CLI.host.remove_all_hosts()
        CC.CLI.host.add_default_host()
        CC.CLI.host.add_host_json("host2")
        CC.CLI.host.add_host_json("host3")
        CC.CLI.host.add_host_json("host4")
        CC.CLI.host.add_host_json("host5")
    end
end

Given (/^(?:|that )there are no hosts added$/) do
    CC.CLI.host.remove_all_hosts()
end

Given (/^(?:|that )only the default host is added$/) do
    CC.CLI.host.ensure_only_default_host_exists()
end

When (/^I am on the hosts page$/) do
    visitHostsPage()
end

When (/^I fill in the Host field with "(.*?)"$/) do |hostName|
    fillInHost(hostName)
end

When (/^I fill in the Port field with "(.*?)"$/) do |rpcPort|
    fillInPort(rpcPort)
end

When (/^I fill in the Resource Pool field with "(.*?)"$/) do |resourcePool|
    fillInResourcePool(resourcePool)
end

When (/^I fill in the RAM Limit field with "(.*?)"$/) do |ramLimit|
    fillInRAMLimit(ramLimit)
end

When (/^I click the add Host button$/) do
    clickAddHostButton()
end

When (/^I click the Hosts Map button$/) do
    CC.UI.HostsPage.hostsMap_button.click()
end

When (/^I add the "(.*?)" host$/) do |host|
    CC.CLI.host.add_host_json(host)
end

Then (/^the "Active" column should be sorted with active hosts on (top|the bottom)$/) do |order|
    list = CC.UI.HostsPage.active_icons
    for i in 0..(list.size - 2)
        if order == "top"
             # assuming + (good ng-scope) before - (down ng-scope) before ! (bad ng-scope)
            list[i][:class].should >= list[i + 1][:class]
        else
            list[i][:class].should <= list[i + 1][:class]        # assuming ! before - before +
        end
    end
end

Then (/^I should see the Add Host dialog$/) do
    CC.UI.HostsPage.addHost_dialog.visible?
end

Then (/^I should see the Host field$/) do
    CC.UI.HostsPage.hostHost_input.visible?
end

Then (/^I should see the Port field$/) do
    CC.UI.HostsPage.rpcPort_input.visible?
end

Then (/^I should see the Resource Pool ID field$/) do
    CC.UI.HostsPage.resourcePool_input.visible?
end

Then (/^I should see the RAM Limit field$/) do
    CC.UI.HostsPage.ramLimit_input.visible?
end

Then (/^I should see an empty Hosts page$/) do
    expect(CC.UI.HostsPage).to have_no_host_entry
    expect(CC.UI.HostsPage).to have_content("0 Results")
    expect(CC.UI.HostsPage).to have_content("No Data Found")
end

Then (/^the Port field should be flagged as invalid$/) do
    expect(CC.UI.HostsPage.rpcPort_input[:class]).to include("ng-invalid")
end


def visitHostsPage()
    oldWait = setDefaultWaitTime(180)
    # printf "....load host page..."
    CC.UI.HostsPage.load
    # printf "done\n....wait for page..."
    expect(CC.UI.HostsPage).to be_displayed
    # printf "done\n....wait for animation..."
    setDefaultWaitTime(oldWait)

    # wait till loading animation clears
    CC.UI.HostsPage.has_no_css?(".loading_wrapper")
    # printf "done\n"
end

def fillInHost(host)
    CC.UI.HostsPage.hostHost_input.set getTableValue(host)
end

def fillInPort(port)
    CC.UI.HostsPage.rpcPort_input.set getTableValue(port)
end

def fillInResourcePool(pool)
    CC.UI.HostsPage.resourcePool_input.select getTableValue(pool)
end

def fillInRAMLimit(commitment)
    CC.UI.HostsPage.ramLimit_input.set getTableValue(commitment)
end

def clickAddHostButton()
    CC.UI.HostsPage.addHost_button.click
    # wait till modal is done loading
    expect(CC.UI.HostsPage).to have_no_css(".uilock", :visible => true)
end

def addHostUI(name, port, pool, commitment)
    clickAddHostButton()
    fillInHost(name)
    fillInPort(port)
    fillInResourcePool(pool)
    fillInRAMLimit(commitment)
    click_link_or_button("Add Host")
end

