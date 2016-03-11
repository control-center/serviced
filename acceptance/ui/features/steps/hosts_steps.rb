Given (/^(?:|that )multiple hosts have been added$/) do
    visitHostsPage()
    @hosts_page.wait_for_host_entries(getDefaultWaitTime())
    if @hosts_page.host_entries.size < 5
        removeAllHostsCLI()
        addDefaultHost()
        addHostJson("host2")
        addHostJson("host3")
        addHostJson("host4")
        addHostJson("host5")
    end
end

Given (/^(?:|that )there are no hosts added$/) do
    removeAllHostsCLI()
end

Given (/^(?:|that )only the default host is added$/) do
    visitHostsPage()
    if (page.has_no_content?("Showing 1 Result") || isNotInRows("table://hosts/defaultHost/name"))
        removeAllHostsCLI()
        addDefaultHost()
    end
end

When (/^I am on the hosts page$/) do
    visitHostsPage()
end

When (/^I fill in the Host Name field with "(.*?)"$/) do |hostName|
    fillInHostAndPort(hostName)
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
    @hosts_page.hostsMap_button.click()
end

When (/^I add the "(.*?)" host$/) do |host|
    addHostJson(host)
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

Then (/^I should see the Add Host dialog$/) do
    @hosts_page.addHost_dialog.visible?
end

Then (/^I should see the Host and port field$/) do
    @hosts_page.hostName_input.visible?
end

Then (/^I should see the Resource Pool ID field$/) do
    @hosts_page.resourcePool_input.visible?
end

Then (/^I should see the RAM Limit field$/) do
    @hosts_page.ramLimit_input.visible?
end

Then (/^I should see an empty Hosts page$/) do
    expect(@hosts_page).to have_no_host_entry
    @hosts_page.assert_text("Showing 0 Results")
    @hosts_page.assert_text("No Data Found")
end

Then (/^the Host and port field should be flagged as invalid$/) do
    expect(@hosts_page.hostName_input[:class]).to include("ng-invalid")
end


def visitHostsPage()
    oldWait = setDefaultWaitTime(180)
    @hosts_page = Hosts.new
    @hosts_page.load
    expect(@hosts_page).to be_displayed
    setDefaultWaitTime(oldWait)

    # wait till loading animation clears
    @hosts_page.has_no_css?(".loading_wrapper")
end

def fillInHostAndPort(host)
    if @hosts_page == nil
         @hosts_page = Hosts.new
    end
    @hosts_page.hostName_input.set getTableValue(host)
end

def fillInResourcePool(pool)
    if @hosts_page == nil
         @hosts_page = Hosts.new
    end
    @hosts_page.resourcePool_input.select getTableValue(pool)
end

def fillInRAMLimit(commitment)
    if @hosts_page == nil
         @hosts_page = Hosts.new
    end
    @hosts_page.ramLimit_input.set getTableValue(commitment)
end

def clickAddHostButton()
    @hosts_page.addHost_button.click
end

def addHost(name, pool, commitment, hostID)
    addHostCLI(name, pool, commitment, hostID)
end

def addHostUI(name, pool, commitment)
    clickAddHostButton()
    fillInHostAndPort(name)
    fillInResourcePool(pool)
    fillInRAMLimit(commitment)
    click_link_or_button("Add Host")
end

def addHostCLI(name, pool, commitment, hostID)
    servicedCLI = getServicedCLI()
    nameValue =  getTableValue(name)
    poolValue =  getTableValue(pool)
    commitmentValue =  getTableValue(commitment)
    cmd = "#{servicedCLI} host add '#{nameValue}' '#{poolValue}' --memory '#{commitmentValue}' 2>&1"

    result = `#{cmd}`

    hostIDValue =  getTableValue(hostID)
    verifyCLIExitSuccess($?, result)
    expect(result.strip).to eq(hostIDValue.to_s)

    refreshPage()
end

def addDefaultHost()
    addHostJson("defaultHost")
end

def addHostJson(host)
    nameAndPort = "table://hosts/" + host + "/nameAndPort"
    pool = "table://hosts/" + host + "/pool"
    commitment = "table://hosts/" + host + "/commitment"
    hostID = "table://hosts/" + host + "/hostID"

    addHost(nameAndPort, pool, commitment, hostID)
end

def removeAllHostsCLI()
    servicedCLI = getServicedCLI()
    cmd = "#{servicedCLI} host list --show-fields ID 2>&1 | grep -v ^ID | xargs --no-run-if-empty #{servicedCLI} host rm 2>&1"
    result = `#{cmd}`
    verifyCLIExitSuccess($?, result)

    # verify all of the hosts were really removed
    cmd = "#{servicedCLI} host list 2>&1"
    result = `#{cmd}`
    verifyCLIExitSuccess($?, result)
    expect(result).to include("no hosts found")

    refreshPage()
end
