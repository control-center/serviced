Given (/^(?:|that )multiple resource pools have been added$/) do
    visitPoolsPage()
    CC.UI.PoolsPage.wait_for_pool_names(getDefaultWaitTime())
    if CC.UI.PoolsPage.pool_names.size < 4
        CC.CLI.pool.remove_all_resource_pools_except_default()
        CC.CLI.pool.add_pool_json("pool2")
        CC.CLI.pool.add_pool_json("pool3")
        CC.CLI.pool.add_pool_json("pool4")
    end
end

# Note this step definition is optimized to use the CLI exclusively so that it can be called before user login
Given (/^(?:|that )the default resource pool is added$/) do
    if (!CC.CLI.pool.check_pool_exists("default"))
        CC.CLI.pool.add_default_pool()
    end
end

Given (/^(?:|that )only the default resource pool is added$/) do
    visitPoolsPage()
    if (page.has_no_content?("1 Result") || isNotInRows("default"))
        CC.CLI.pool.remove_all_resource_pools_except_default()
    end
end

# Note this step definition is optimized to use the CLI exclusively so that it can be called before user login
Given (/^(?:|that )the "(.*?)" pool is added$/) do |pool|
    if (!CC.CLI.pool.check_pool_exists(pool))
        CC.CLI.pool.add_pool(pool, "added for tests")
    end
end

Given (/^(?:|that )the "(.*?)" virtual IP is added to the "(.*?)" pool$/) do |ip, pool|
    visitPoolsPage()
    if (isNotInRows(pool))
        CC.CLI.pool.add_pool(pool, "added for virtual IP")
    end
    viewDetails(pool, "pools")
    if (isNotInRows("table://virtualips/" + ip + "/ip"))
        addVirtualIpJson(ip)
    end
end

Given (/^(?:|that )the "(.*?)" pool has no virtual IPs$/) do |pool|
    visitPoolsPage()
    if (isNotInRows(pool))
        CC.CLI.pool.add_pool(pool, "added for no virtual IPs")
    else
        viewDetails(pool, "pools")
        if (CC.UI.PoolsPage.virtualIps_table.has_no_text?("No Data Found"))
            removeAllEntries("address")
        end
    end
end

When (/^I am on the resource pool page$/) do
    visitPoolsPage()
end

When (/^I remove all resource pools$/) do
    CC.CLI.pool.remove_all_resource_pools_except_default()
end

When (/^I click the add Resource Pool button$/) do
    clickAddPoolButton()
end

When (/^I fill in the Resource Pool name field with "(.*?)"$/) do |resourcePool|
    fillInResourcePoolField(resourcePool)
end

When (/^I fill in the Description field with "(.*?)"$/) do |description|
    fillInDescriptionField(description)
end

When (/^I add the "(.*?)" pool$/) do |pool|
    CC.CLI.pool.add_pool_json(pool)
end

When (/^I click the Add Virtual IP button$/) do
    clickAddVirtualIpButton()
    # wait till modal is done loading
    expect(CC.UI.PoolsPage).to have_no_css(".uilock", :visible => true)
end

When (/^I add the virtual IP$/) do
    addVirtualIpButton()
end

When (/^I fill in the IP field with "(.*?)"$/) do |ip|
    fillInIpField(ip)
end

When (/^I fill in the Netmask field with "(.*?)"$/) do |netmask|
    fillInNetmaskField(netmask)
end

When (/^I fill in the Interface field with "(.*?)"$/) do |interface|
    fillInInterfaceField(interface)
end

Then (/^I should see the add Resource Pool button$/) do
    CC.UI.PoolsPage.addPool_button.visible?
end

Then (/^I should see the Resource Pool name field$/) do
    CC.UI.PoolsPage.poolName_input.visible?
end

Then (/^I should see the Description field$/) do
    CC.UI.PoolsPage.description_input.visible?
end

Then (/^I should see the IP field$/) do
    CC.UI.PoolsPage.ip_input.visible?
end

Then (/^I should see the Netmask field$/) do
    CC.UI.PoolsPage.netmask_input.visible?
end

Then (/^I should see the Interface field$/) do
    CC.UI.PoolsPage.interface_input.visible?
end

Then (/^the "(.*)" button should (not )?be disabled$/) do |name, enabled|
    button = find(:xpath, "//button[@name='" + getTableValue(name) + "']")
    if (enabled)
        not button.has_css?('disabled')
    else
        button.has_css?('disabled')
    end
end

def visitPoolsPage()
    oldWait = setDefaultWaitTime(180)
    CC.UI.PoolsPage.navbar.resourcePools.click()
    expect(CC.UI.PoolsPage).to be_displayed
    setDefaultWaitTime(oldWait)

    # wait till loading animation clears
    CC.UI.PoolsPage.has_no_css?(".loading_wrapper")
end

def clickAddPoolButton()
    CC.UI.PoolsPage.addPool_button.click()
    # wait till modal is done loading
    expect(CC.UI.PoolsPage).to have_no_css(".uilock", :visible => true)
end

def fillInResourcePoolField(name)
    val = getTableValue(name)
    CC.UI.PoolsPage.poolName_input.set val
    expect(CC.UI.PoolsPage.poolName_input.value).to eq val
end

def fillInDescriptionField(description)
    val = getTableValue(description)
    CC.UI.PoolsPage.description_input.set val
    expect(CC.UI.PoolsPage.description_input.value).to eq val
end

def clickAddVirtualIpButton()
    CC.UI.PoolsPage.addVirtualIp_button.click()
end

def fillInIpField(address)
    val = getTableValue(address)
    CC.UI.PoolsPage.ip_input.set val
    expect(CC.UI.PoolsPage.ip_input.value).to eq val
end

def fillInNetmaskField(netmask)
    val = getTableValue(netmask)
    CC.UI.PoolsPage.netmask_input.set val
    expect(CC.UI.PoolsPage.netmask_input.value).to eq val
end

def fillInInterfaceField(interface)
    val = getTableValue(interface)
    CC.UI.PoolsPage.interface_input.set val
    expect(CC.UI.PoolsPage.interface_input.value).to eq val
end

def addVirtualIpButton()
    CC.UI.PoolsPage.dialogAddVirtualIp_button.click()
end

def addVirtualIp(ip, netmask, interface)
    clickAddVirtualIpButton()
    fillInIpField(ip)
    fillInNetmaskField(netmask)
    fillInInterfaceField(interface)
    addVirtualIpButton()
end

def addVirtualIpJson(ip)
    addVirtualIp("table://virtualips/" + ip + "/ip", "table://virtualips/" + ip + "/netmask",
        "table://virtualips/" + ip + "/interface")
end

def addPoolUI(name, description)
    clickAddPoolButton()
    fillInResourcePoolField(name)
    fillInDescriptionField(description)
    click_link_or_button("Add Resource Pool")
end
