DEFAULT_HOST = "172.17.42.1:4979"
DEFAULT_POOL = "default"
DEFAULT_COMMITMENT = "50%"

Given(/^there are no hosts defined$/) do
  visitHostsPage()
  removeAllHosts()
end

Given(/^only the default host is defined$/) do
  visitHostsPage()
  removeAllHosts()
  clickAddHostButton()
  fillInHostAndPort(DEFAULT_HOST)
  fillInResourcePool(DEFAULT_POOL)
  fillInRAMCommitment(DEFAULT_COMMITMENT)
  click_link_or_button("Add Host")
end

When(/^I am on the hosts page$/) do
    visitHostsPage()
end

When(/^I fill in the Host Name field with the default host name$/) do
    fillInHostAndPort(DEFAULT_HOST)
end

When(/^I fill in the Resource Pool field with the default resource pool$/) do
    fillInResourcePool(DEFAULT_POOL)
end

When(/^I fill in the RAM Commitment field with the default RAM commitment$/) do
    fillInRAMCommitment(DEFAULT_COMMITMENT)
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
  list = page.all("[ng-if$='host.active']")
  for i in 0..(list.size - 2)
    if order == "top"
       # assuming + (good ng-scope) before - (down ng-scope) before ! (bad ng-scope)
      list[i][:class].should >= list[i + 1][:class]
    else
      list[i][:class].should <= list[i + 1][:class]    # assuming ! before - before +
    end
  end
end

Then /^I should see the Add Host dialog$/ do
    @hosts_page.assert_selector '.modal-content'        # class for dialog box
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
    @hosts_page.assert_no_selector("[ng-repeat='host in $data']")
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
    #        even though the same syntax works on FF
    # @hosts_page.load
    # expect(@hosts_page).to be_displayed
    within(".navbar-collapse") do
        click_link("Hosts")
    end
    expect(@hosts_page).to be_displayed
end

def fillInHostAndPort(host)
    @hosts_page.hostName_input.set host
end

def fillInResourcePool(pool)
    @hosts_page.resourcePool_input.select pool
end

def fillInRAMCommitment(commitment)
    @hosts_page.ramCommitment_input.set commitment
end

def clickAddHostButton()
    @hosts_page.addHost_button.click
end

def removeAllHosts()
    defaultMatch = Capybara.match
    Capybara.match=:first
    while @hosts_page.all("[ng-repeat='host in $data']").size != 0 do
      click_link_or_button("Delete")
      click_link_or_button("Remove Host")
    end
    Capybara.match = defaultMatch
end