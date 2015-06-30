When(/^I am on the resource pool page$/) do
    visitPoolsPage()
end

When /^I click the add Resource Pool button$/ do
    @pools_page.addPool_button.click()
end

When(/^I fill in the Resource Pool name field with "(.*?)"$/) do |resourcePool|
    @pools_page.poolName_input.set resourcePool
end

When(/^I fill in the Description field with "(.*?)"$/) do |description|
    @pools_page.description_input.set description
end

Then /^I should see the add Resource Pool button$/ do
    @pools_page.addPool_button.visible?
end

Then /^I should see the Resource Pool name field$/ do
    @pools_page.poolName_input.visible?
end

Then /^I should see the Description field$/ do
    @pools_page.description_input.visible?
end

def visitPoolsPage()
    @pools_page = Pools.new
    within(".navbar-collapse") do
        click_link("Resource Pools")
    end
    expect(@pools_page).to be_displayed
end