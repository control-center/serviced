Given (/^(?:|that )the admin user is logged in$/) do
    loginAsDefaultUser()
end

When (/^I fill in "([^"]*)" with "([^"]*)"$/) do |element, text|
    entry = getTableValue(text)
    fill_in element, with: entry
end

When (/^I fill in the "([^"]*)" field with "(.*?)"$/) do |field, text|
    find(field).set(getTableValue(text))
end

When (/^I click (?:|on )"([^"]*)"$/) do |text|
    click_link_or_button(getTableValue(text))
end

When (/^I close the dialog$/) do
    closeDialog()
end

When (/^I remove "([^"]*)"$/) do |name|
    entry = getTableValue(name)
    within("tr[class='ng-scope']", :text => entry) do
        click_link_or_button("Delete")
    end
end

When (/^I select "(.*?)"$/) do |name|
    selectOption(name)
end

When (/^I sort by "([^"]*)" in ([^"]*) order$/) do |category, order|
    sortColumn(category, order)
end

When (/^I view the details (?:for|of) "(.*?)" in the "(.*?)" table$/) do |name, table|
    viewDetails(name, getTableType(table))
end

When (/^I hover over the "(.*?)" graph$/) do |graph|
    hoverOver(graph)
end

Then (/^I should see "(.*?)"$/) do |text|
    expect(page).to have_content getTableValue(text)
end

Then (/^I should see "(.*?)" after waiting no more than "(.*?)" seconds$/) do |text, time|
    expect(page).to have_content(getTableValue(text), wait: time.to_f)
end

Then (/^I should not see "(.*?)"$/) do |text|
    expect(page).to have_no_content getTableValue(text)
end

Then (/^I should see the "([^"]*)"$/) do |element|
    find(element).visible?
end

Then (/^I should see( the sum of)? "(.*?)" in the "([^"]*)" column$/) do |sum, text, column|
    text = getSum(text) if sum
    expect(isInColumn(text, column)).to be true
end

Then (/^I should not see "(.*?)" in the "([^"]*)" column$/) do |text, column|
    expect(isNotInColumn(text, column)).to be true
end

Then (/^the "([^"]*)" column should be sorted in ([^"]*) order$/) do |category, order|
    if order == "ascending"
        assertSortedColumn(category, true)
    else
        assertSortedColumn(category, false)
    end
end

Then (/^I should see an entry for "(.*?)" in the table$/) do |row|
    expect(isInRows(row)).to be true
end

Then (/^I should not see an entry for "(.*?)" in the table$/) do |row|
    expect(isNotInRows(row)).to be true
end

Then (/^I should see "(.*?)" in the "(.*?)" graph$/) do |text, graph|
    within(page.find("div[class='zenchartContainer']", :text => graph)) do
        expect(page).to have_content(getTableValue(text))
    end
end

Then (/^I should see "(.*?)" in the hover box$/) do |text|
    within(page.find("div[class^='nvtooltip']")) do
        expect(page).to have_content(getTableValue(text))
    end
end

Then (/^the details for "(.*?)" should be( the sum of)? "(.*?)"$/) do |header, sum, text|
    text = getSum(text) if sum
    expect(checkDetails(text, header)).to be true
end

Then (/^I should see "(.*?)" in the "(.*?)" table$/) do |text, table|
    table = getTableType(table)
    within(page.find("table[data-config='#{table}Table']")) do
        expect(page).to have_content(getTableValue(text))
    end
end

Then (/^I should not see "(.*?)" in the "(.*?)" table$/) do |text, table|
    table = getTableType(table)
    within(page.find("table[data-config='#{table}Table']")) do
        expect(page).to have_no_content(getTableValue(text))
    end
end


def getTableType(table)
    type = ""
    if table == "Applications" || table == "Services"
        type = "services"
    elsif table == "Application Templates" || table == "Templates"
        type = "templates"
    elsif table == "Hosts"
        type = "hosts"
    elsif table == "Resource Pools"
        type = "pools"
    end
    return type
end

def viewDetails(name, table)
    name = getTableValue(name)
    within(page.find("table[data-config='#{table}Table']")) do
        page.find("[ng-click]", :text => name).click()
    end
end

def assertSortedColumn(category, order)
    list = page.all("td[data-title-text='#{category}'][sortable]")
    for i in 0..(list.size - 2)
        if category == "Created" || category == "Last Modified"
            if order
                expect(DateTime.parse(list[i].text)).to be <= DateTime.parse(list[i + 1].text)
            else
                expect(DateTime.parse(list[i].text)).to be >= DateTime.parse(list[i + 1].text)
            end
        elsif category == "Memory"
            if order
                expect(list[i].text[0..-4].to_f).to be <= list[i + 1].text[0..-4].to_f
            else
                expect(list[i].text[0..-4].to_f).to be >= list[i + 1].text[0..-4].to_f
            end
        elsif category == "CPU Cores"
            if order
                expect(list[i].text.to_i).to be <= list[i + 1].text.to_i
            else
                expect(list[i].text.to_i).to be >= list[i + 1].text.to_i
            end
        else
            if order
            # Category sorting ignores case
                expect(list[i].text.downcase).to be <= list[i + 1].text.downcase
            else
                expect(list[i].text.downcase).to be >= list[i + 1].text.downcase
            end
        end
    end
end

def getSum(urls)
    urlList = urls.split(", ")
    sum = 0
    for i in 0..(urlList.size - 1)
        sum += getTableValue(urlList[i]).to_i
    end
    return sum.to_s
end

def hoverOver(graph)
    page.find("div[class='zenchartContainer']", :text => graph).hover()
end

def isInRows(row)
    found = false
    name = getTableValue(row)
    found = page.has_css?("tr[ng-repeat$='in $data']", :text => name)
    return found
end

def isNotInRows(row)
    notFound = false
    name = getTableValue(row)
    notFound = page.has_no_css?("tr[ng-repeat$='in $data']", :text => name)
    return notFound
end

def isInColumn(text, column)
    # attribute that includes name of column of all table cells
    hasEntry = false
    cell = getTableValue(text).to_s
    hasEntry = true if page.has_css?("td[data-title-text='#{column}']", :text => cell)
    return hasEntry
end

def isNotInColumn(text, column)
    # attribute that includes name of column of all table cells
    hasNoEntry = false
    cell = getTableValue(text).to_s
    hasNoEntry = true if page.has_no_css?("td[data-title-text='#{column}']", :text => cell)
    return hasNoEntry
end

def checkDetails(detail, header)
    found = false
    detail = getTableValue(detail)
    within(page.find("div[class='vertical-info']", :text => header)) do
        found = true if page.has_text?(detail)
        expect(page).to have_content(detail)
    end
    return found
end

def closeDialog()
    page.find("button[class^='close glyphicon']").click()
end

def sortColumn(category, sortOrder)
    if sortOrder == "ascending"
        order = 'header  sortable sort-asc'
    else
        order = 'header  sortable sort-desc'
    end

    # click until column header shows ascending/descending
    categoryLink = page.find("[class^='header  sortable']", :text => /\A#{category}\z/)
    while categoryLink[:class] != order do
        categoryLink.click()
        categoryLink = page.find("[class^='header  sortable']", :text => /\A#{category}\z/)
    end
end

def selectOption(name)
    entry = getTableValue(name)
    within("tr[class='clickable ng-scope']", :text => entry) do
        page.find("input[type='radio']").click()
    end
end

def removeEntry(name, category)
    name = getTableValue(name)
    within(page.find("tr[ng-repeat='#{category} in $data']", :text => name, match: :first)) do
        click_link_or_button("Delete")
    end
    click_link_or_button("Remove")
    refreshPage()
end

def removeAllEntries(category)
    entry = ""
    while (page.has_css?("tr[ng-repeat='#{category} in $data']", :text => "Delete")) do
        within(page.find("tr[ng-repeat='#{category} in $data']", :text => "Delete", match: :first)) do
            entry = page.text
            click_link_or_button("Delete")
        end
        click_link_or_button("Remove")
        refreshPage() if category == "service"
        expect(page).to have_no_content(entry)
    end
end

def refreshPage()
    visit page.driver.browser.current_url
end

def getTableValue(valueOrTableUrl)
    if valueOrTableUrl.start_with?("table://") == false
        return valueOrTableUrl
    end
    parsedUrl = valueOrTableUrl.split(/\W+/)
    if parsedUrl.size != 4
        raise(ArgumentError.new('Invalid URL'))
    end

    tableType = parsedUrl[1]
    tableName = parsedUrl[2]
    propertyName = parsedUrl[3]
    if PARSED_DATA[tableType].nil?
        raise(ArgumentError.new('Invalid table type'))
    elsif PARSED_DATA[tableType][tableName].nil?
        raise(ArgumentError.new('Invalid table name'))
    elsif PARSED_DATA[tableType][tableName][propertyName].nil?
        raise(ArgumentError.new('Invalid property name'))
    else
        data = PARSED_DATA[tableType][tableName][propertyName]
        if data.to_s.include? "%{local_ip}"
            data.sub! "%{local_ip}", HOST_IP
        end
        if data.to_s.include? "%{target_host}"
            data.sub! "%{target_host}", TARGET_HOST
        end
        return data
    end
end

def getServicedCLI()
    return "/capybara/serviced --endpoint #{TARGET_HOST}:4979"
end

#
# Login if not already logged in. 
#
# Note that by saving cookies after the first successful login and restoring them
#   on subsequent calls (thus bypassing the need to login), dramatically improves performance for all
#   test cases.
def loginAsDefaultUser()

  # Before we can set cookies, we must visit a page first
  visitLoginPage()

  #
  # FIXME: CC-2129
  # Apparently, there is a bug in $ngCookies which sometimes causes the session cookies not to be loaded.
  # Specifically, checkLogin() in authService.js does not see the restored cookie values (sometimes) even though
  # they are there. If that problem is resolved (either by upgrading Angular or writing our own implementation
  # of something like $ngCookies), then uncommenting this code will result in significant improvement in the
  # performance of acceptance tests because we can skip most logins.
  #
  # # if we have cookies from a previous login, restore them and we're done
  # if $saved_cookies != nil
  #   # printf "saved cookies=%s\n", $saved_cookies.inspect
  #   $saved_cookies.each do |cookie|
  #     page.driver.browser.manage.add_cookie(
  #       {name: cookie[:name], value: cookie[:value]}
  #     )
  #   end
  #
  #   # simulate starting on the default landing page; otherwise we're still sitting on the login page
  #   visitDefaultPage()
  #   return
  # end

  fillInDefaultUserID()
  fillInDefaultPassword()
  clickSignInButton()

  # login redirects to application page, but
  # deploy wizard may appears, so automatically
  # close it
  closeDeployWizard()

  # FIXME: CC-2129
  # $saved_cookies = page.driver.browser.manage.all_cookies
  # printf "saving cookies=%s\n", $saved_cookies.inspect
end

#
# Verify that CLI exit status is 0.
# If it fails, include the command output in the error message.
#
def verifyCLIExitSuccess(processStatus, output)
    errorMsg = "CLI return code %d is not 0. Command Output=%s" % [processStatus.exitstatus, output]
    expect(processStatus.exitstatus).to eq(0), errorMsg
end

def getDefaultWaitTime()
    return Capybara.default_max_wait_time
end

def setDefaultWaitTime(newWait)
    oldWait = Capybara.default_max_wait_time
    Capybara.default_max_wait_time = newWait
    return oldWait
end

def visitDefaultPage()
  visitApplicationsPage()
end
