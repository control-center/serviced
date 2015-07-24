Given /^that multiple resource pools have been added$/ do
    visitPoolsPage()
    waitForPageLoad()
    if @pools_page.pool_entries.size < 4
        removeAllPools()
        waitForPageLoad()
        addDefaultPool()
        addPoolJson("pool2")
        addPoolJson("pool3")
        addPoolJson("pool4")
        expect(checkRows("table://pools/pool2/name")).to be true
        expect(checkRows("table://pools/pool4/name")).to be true
    end
end

Given /^that the default resource pool exists$/ do
    visitPoolsPage()
    hasDefault = checkRows("default")
    if (hasDefault == false)
        addDefaultPool()
    end
end

Given /^that only the default resource pool exists$/ do
    visitPoolsPage()
    if (!page.has_content?("Showing 1 Result") || !checkRows("default"))
        removeAllPools()
        addDefaultPool()
    end
end

Given /^that the "(.*?)" pool exists$/ do |pool|
    visitPoolsPage()
    if (checkRows(pool) == false)
        addPool(pool, "added for tests")
    end
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

When(/^I add the "(.*?)" pool$/) do |pool|
    addPoolJson(pool)
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
    waitForPageLoad()
end

def addDefaultPool()
    addPoolJson("defaultPool")
end

def addPoolJson(pool)
    addPool("table://pools/" + pool + "/name", "table://pools/" + pool + "/description")
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