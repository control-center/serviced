When(/^I am on the hosts page$/) do
    visitHostsPage()
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


=begin
Then /^The hosts should be sorted by "(.*?)"$/ do |sort|
    page.find("[class^='header  sortable']", :text => /\A#{category}\z/).click()
    page.body.index(new_comment.text).should < page.body.index(old_comment.text)
end
=end

=begin
bad !host.resourcesGood() 3->31.37, e->0
ng-binding ng-hide 3->3s
ng-binding e->31.37 shown, no hide
=end


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