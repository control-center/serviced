Given(/^that multiple hosts have been added$/) do
    visitHostsPage()
    if @hosts_page.has_text?("Showing 0 Results") || @hosts_page.has_text?("Showing 1 Result")
        removeAllEntries("host")
        addDefaultHost()
        addHost("table://hosts/host2/nameAndPort", "table://hosts/host2/pool", \
            "table://hosts/host2/commitment")
        addHost("table://hosts/host3/nameAndPort", "table://hosts/host3/pool", \
            "table://hosts/host3/commitment")
        checkRows("table://hosts/defaultHost/name", true)
        checkRows("table://hosts/host2/name", true)
        checkRows("table://hosts/host3/name", true)
    end
end

Given(/^there are no hosts defined$/) do
    visitHostsPage()
    removeAllEntries("host")
end

Given(/^only the default host is defined$/) do
    visitHostsPage()
    removeAllEntries("host")
    addDefaultHost()
end

When(/^I am on the hosts page$/) do
    visitHostsPage()
end

When(/^I fill in the Host Name field with "(.*?)"$/) do |hostName|
    fillInHostAndPort(hostName)
end

When(/^I fill in the Resource Pool field with "(.*?)"$/) do |resourcePool|
    fillInResourcePool(resourcePool)
end

When(/^I fill in the RAM Commitment field with "(.*?)"$/) do |ramCommitment|
    fillInRAMCommitment(ramCommitment)
end

When /^I click the Add-Host button$/ do
    clickAddHostButton()
end

Then (/^the "Active" column should be sorted with active hosts on (top|the bottom)$/) do |order|
    list = @hosts_page.active_icons
    for i in 0..(list.size - 2)
        if order == "top"
             # assuming + (good ng-scope) before - (down ng-scope) before ! (bad ng-scope)
            list[i][:class].should >= list[i + 1][:class]
        else
            list[i][:class].should <= list[i + 1][:class]        # assuming ! before - before +
        end
    end
end

Then /^I should see the Add Host dialog$/ do
    @hosts_page.addHost_dialog.visible?
end

Then /^I should see the Host and port field$/ do
    @hosts_page.hostName_input.visible?
end

Then /^I should see the Resource Pool ID field$/ do
    @hosts_page.resourcePool_input.visible?
end

Then /^I should see the RAM Commitment field$/ do
    @hosts_page.ramCommitment_input.visible?
end

Then /^I should see an empty Hosts page$/ do
    expect(@hosts_page).to have_no_host_entry
    @hosts_page.assert_text("Showing 0 Results")
    @hosts_page.assert_text("No Data Found")
end

Then (/^the Host and port field should be flagged as invalid$/) do
    expect(@hosts_page.hostName_input[:class]).to include("ng-invalid")
end


def visitHostsPage()
    @hosts_page = Hosts.new
    #
    # FIXME: For some reason the following load fails on Chrome for this page,
    #                even though the same syntax works on FF
    # @hosts_page.load
    # expect(@hosts_page).to be_displayed
    @hosts_page.navbar.hosts.click()
    expect(@hosts_page).to be_displayed
end

def fillInHostAndPort(host)
    @hosts_page.hostName_input.set getTableValue(host)
end

def fillInResourcePool(pool)
    @hosts_page.resourcePool_input.select getTableValue(pool)
end

def fillInRAMCommitment(commitment)
    @hosts_page.ramCommitment_input.set getTableValue(commitment)
end

def clickAddHostButton()
    @hosts_page.addHost_button.click
end

def addHost(name, pool, commitment)
    clickAddHostButton()
    fillInHostAndPort(name)
    fillInResourcePool(pool)
    fillInRAMCommitment(commitment)
    click_link_or_button("Add Host")
end

def addDefaultHost()
    addHost("table://hosts/defaultHost/nameAndPort", \
        "table://hosts/defaultHost/pool", \
        "table://hosts/defaultHost/commitment")
end