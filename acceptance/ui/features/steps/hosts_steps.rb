When(/^I am on the hosts page$/) do
    visitHostsPage()
end

When(/^I fill in the Host Name field with the default host name$/) do
    fillInDefaultHostAndPort()
end

When(/^I fill in the Resource Pool field with the default resource pool$/) do
    fillInDefaultResourcePool()
end

When(/^I fill in the RAM Commitment field with the default RAM commitment$/) do
    fillInDefaultRAMCommitment()
end

When(/^I fill in the Host Name field with "(.*?)"$/) do |hostName|
    @hosts_page.hostName_input.set hostName
end

When(/^I fill in the Resource Pool field with "(.*?)"$/) do |resourcePool|
    @hosts_page.resourcePool_input.select resourcePool
end

When(/^I fill in the RAM Commitment field with "(.*?)"$/) do |ramCommitment|
    @hosts_page.ramCommitment_input.set ramCommitment
end

When /^I click the Add-Host button$/ do
    clickAddHostButton()
end

Then (/^the "Active" column should be sorted with active hosts on (top|the bottom)$/) do |order|
  list = page.all("[ng-if$='host.active']")
  for i in 0..(list.size - 2)
    if order == "top"
       # assuming ! (bad ng-scope) before - (down ng-scope) before + (good ng-scope)
      list[i][:class] <= list[i + 1][:class]
    else
      list[i][:class] >= list[i + 1][:class]    # assuming + before - before !
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

def clickAddHostButton()
    @hosts_page.addHost_button.click
end

def fillInDefaultHostAndPort()
    @hosts_page.hostName_input.set "172.17.42.1:4979"
end

def fillInDefaultResourcePool()
    @hosts_page.resourcePool_input.select "default"
end

def fillInDefaultRAMCommitment()
    @hosts_page.ramCommitment_input.set "50%"
end

def removeAllHosts()
    Capybara.match=:first
    while page.all("[ng-repeat='host in $data']").size != 0 do
      click_link_or_button("Delete")
      click_link_or_button("Remove Host")
    end
    Capybara.match=:smart
end