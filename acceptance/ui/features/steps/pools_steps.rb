Given /^that multiple resource pools have been added$/ do
    visitPoolsPage()
    if @pools_page.pool_entries.size < 2
        removeAllPools()
        addDefaultPool()
        addPool("table://pools/pool2/name", "table://pools/pool2/description")
        checkRows("table://pools/defaultPool/name", true)
        checkRows("table://pools/pool2/name", true)
    end
end

Given /^that only the default resource pool is defined$/ do
    visitPoolsPage()
    removeAllPools()
    addDefaultPool()
end

When(/^I am on the resource pool page$/) do
    visitPoolsPage()
end

When /^I click the add Resource Pool button$/ do
    clickAddPoolButton()
end

When(/^I fill in the Resource Pool name field with "(.*?)"$/) do |resourcePool|
    fillInResourcePoolField(resourcePool)
end

When(/^I fill in the Description field with "(.*?)"$/) do |description|
    fillInDescriptionField(description)
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
    @pools_page.navbar.resourcePools.click()
    expect(@pools_page).to be_displayed
end

def clickAddPoolButton()
    @pools_page.addPool_button.click()
end

def fillInResourcePoolField(name)
    @pools_page.poolName_input.set getTableValue(name)
end

def fillInDescriptionField(description)
    @pools_page.description_input.set getTableValue(description)
end

def addPool(name, description)
    clickAddPoolButton()
    fillInResourcePoolField(name)
    fillInDescriptionField(description)
    click_link_or_button("Add Resource Pool")
end

def addDefaultPool()
    addPool("table://pools/defaultPool/name", "table://pools/defaultPool/description")
end

def removeAllPools()
    visitHostsPage()
    removeAllEntries("host")
    visitApplicationsPage()
    removeAllEntries("service")
    removeAllEntries("template")
    visitPoolsPage()
    removeAllEntries("pool")
end