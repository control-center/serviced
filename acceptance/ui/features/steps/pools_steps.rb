Given (/^(?:|that )multiple resource pools have been added$/) do
    visitPoolsPage()
    @pools_page.wait_for_pool_names(Capybara.default_wait_time)
    if @pools_page.pool_names.size < 4
        removeAllPoolsExceptDefault()
        addPoolJson("pool2")
        addPoolJson("pool3")
        addPoolJson("pool4")
    end
end

Given (/^(?:|that )the default resource pool is added$/) do
    visitPoolsPage()
    hasDefault = isInRows("default")
    if (hasDefault == false)
        addDefaultPool()
    end
end

Given (/^(?:|that )only the default resource pool is added$/) do
    visitPoolsPage()
    if (page.has_no_content?("Showing 1 Result") || isNotInRows("default"))
        removeAllPoolsExceptDefault()
    end
end

Given (/^(?:|that )the "(.*?)" pool is added$/) do |pool|
    visitPoolsPage()
    if (isNotInRows(pool))
        addPool(pool, "added for tests")
    end
end

Given (/^(?:|that )the "(.*?)" virtual IP is added to the "(.*?)" pool$/) do |ip, pool|
    visitPoolsPage()
    if (isNotInRows(pool))
        addPool(pool, "added for virtual IP")
    end
    viewDetails(pool, "pools")
    if (isNotInRows("table://virtualips/" + ip + "/ip"))
        addVirtualIpJson(ip)
    end
end

Given (/^(?:|that )the "(.*?)" pool has no virtual IPs$/) do |pool|
    visitPoolsPage()
    if (isNotInRows(pool))
        addPool(pool, "added for no virtual IPs")
    else
        viewDetails(pool, "pools")
        if (@pools_page.virtualIps_table.has_no_text?("No Data Found"))
            removeAllEntries("address")
        end
    end
end

When (/^I am on the resource pool page$/) do
    visitPoolsPage()
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
    addPoolJson(pool)
end

When (/^I click the Add Virtual IP button$/) do
    clickAddVirtualIpButton()
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
    @pools_page.addPool_button.visible?
end

Then (/^I should see the Resource Pool name field$/) do
    @pools_page.poolName_input.visible?
end

Then (/^I should see the Description field$/) do
    @pools_page.description_input.visible?
end

Then (/^I should see the IP field$/) do
    @pools_page.ip_input.visible?
end

Then (/^I should see the Netmask field$/) do
    @pools_page.netmask_input.visible?
end

Then (/^I should see the Interface field$/) do
    @pools_page.interface_input.visible?
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

def clickAddVirtualIpButton()
    @pools_page.addVirtualIp_button.click()
end

def fillInIpField(address)
    @pools_page.ip_input.set getTableValue(address)
end

def fillInNetmaskField(netmask)
    @pools_page.netmask_input.set getTableValue(netmask)
end

def fillInInterfaceField(interface)
    @pools_page.interface_input.set getTableValue(interface)
end

def addVirtualIpButton()
    @pools_page.dialogAddVirtualIp_button.click()
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

def addPool(name, description)
    addPoolCLI(name, description)
end

def addPoolUI(name, description)
    clickAddPoolButton()
    fillInResourcePoolField(name)
    fillInDescriptionField(description)
    click_link_or_button("Add Resource Pool")
end

def addPoolCLI(name, description)
    servicedCLI = getServicedCLI()
    nameValue =  getTableValue(name)
    # description is not used by the CLI
    # descriptionValue =  getTableValue(description)
    cmd = "#{servicedCLI} pool add '#{nameValue}' 2>&1"

    result = `#{cmd}`

    expect($?.exitstatus).to eq(0)
    expect(result.strip).to eq(nameValue.to_s)
end

def addDefaultPool()
    addPoolJson("defaultPool")
end

def addPoolJson(pool)
    addPool("table://pools/" + pool + "/name", "table://pools/" + pool + "/description")
end

def removeAllPoolsExceptDefault()
    visitApplicationsPage()
    removeAllEntries("service")
    removeAllHostsCLI()
    removeAllPoolsCLI()
    addDefaultPool()
end

def removeAllPoolsCLI()
    servicedCLI = getServicedCLI()
    cmd = "#{servicedCLI} pool list --show-fields ID 2>&1 | grep -v ^ID | xargs --no-run-if-empty #{servicedCLI} pool rm 2>&1"
    result = `#{cmd}`
    expect($?.exitstatus).to eq(0)

    # verify all of the hosts were really removed
    cmd = "#{servicedCLI} pool list 2>&1"
    result = `#{cmd}`
    expect($?.exitstatus).to eq(0)
    expect(result).to include("no resource pools found")
end
