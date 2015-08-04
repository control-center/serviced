Given (/^(?:|that )the admin user is logged in$/) do
    visitLoginPage()
    fillInDefaultUserID()
    fillInDefaultPassword()
    clickSignInButton()
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

When (/^I wait for the page to load$/) do
    waitForPageLoad()
end

When (/^I view the details (?:for|of) "(.*?)"$/) do |name|
    viewDetails(name)
end

When (/^I hover over the "(.*?)" graph$/) do |graph|
    hoverOver(graph)
end

Then (/^I should see "(.*?)"$/) do |text|
    expect(page).to have_content getTableValue(text)
end

Then (/^I should not see "(.*?)"$/) do |text|
    expect(page).to have_no_content getTableValue(text)
end

Then (/^I should see the "([^"]*)"$/) do |element|
    find(element).visible?
end

Then (/^I should see( the sum of)? "(.*?)" in the "([^"]*)" column$/) do |sum, text, column|
    text = getSum(text) if sum
    expect(checkColumn(text, column)).to be true
end

Then (/^I should not see "(.*?)" in the "([^"]*)" column$/) do |text, column|
    expect(checkColumn(text, column)).to be false
end

Then (/^the "([^"]*)" column should be sorted in ([^"]*) order$/) do |category, order|
    if order == "ascending"
        assertSortedColumn(category, true)
    else
        assertSortedColumn(category, false)
    end
end

Then (/^I should see an entry for "(.*?)" in the table$/) do |row|
    expect(checkRows(row)).to be true
end

Then (/^I should not see an entry for "(.*?)" in the table$/) do |row|
    expect(checkRows(row)).to be false
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

Then (/^"(.*?)" should be active$/) do |entry|
    expect(checkActive(entry)).to be true
end


def viewDetails(name)
    name = getTableValue(name)
    find("[ng-click]", :text => /\A#{name}\z/).click()
    waitForPageLoad()
end

def checkActive(entry)
    within(page.find("tr", :text => entry)) do
        return page.has_css?("[class*='good']")
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

def checkRows(row)
    waitForPageLoad()
    found = false
    name = getTableValue(row)
    found = page.has_css?("tr[ng-repeat$='in $data']", :text => name)
    return found
end

def checkColumn(text, column)
    # attribute that includes name of column of all table cells
    hasEntry = false
    cell = getTableValue(text).to_s
    hasEntry = true if page.has_css?("tr[ng-repeat$='in $data']", :text => cell)
    return hasEntry
end

def checkDetails(detail, header)
    found = false
    detail = getTableValue(detail)
    within(page.find("div[class='vertical-info']", :text => header)) do
        found = true if page.has_text?(detail)
    end
    return found
end

def closeDialog()
    page.find("button[class^='close glyphicon']").click()
end

def sortColumn(category, sortOrder)
    categoryLink = page.find("[class^='header  sortable']", :text => /\A#{category}\z/)
    if sortOrder == "ascending"
        order = 'header  sortable sort-asc'
    else
        order = 'header  sortable sort-desc'
    end
    # click until column header shows ascending/descending
    while categoryLink[:class] != order do
        categoryLink.click()
    end
end

def selectOption(name)
    entry = getTableValue(name)
    within("tr[class='clickable ng-scope']", :text => entry) do
        page.find("input[type='radio']").click()
    end
end

def removeEntry(name, category)
    waitForPageLoad()
    name = getTableValue(name)
    within(page.find("tr[ng-repeat='#{category} in $data']", :text => name, match: :first)) do
        click_link_or_button("Delete")
    end
    click_link_or_button("Remove")
    refreshPage()
end

def removeAllEntries(category)
    waitForPageLoad()
    while (page.has_css?("tr[ng-repeat='#{category} in $data']", :text => "Delete")) do
        within(page.find("tr[ng-repeat='#{category} in $data']", :text => "Delete", match: :first)) do
            click_link_or_button("Delete")
        end
        click_link_or_button("Remove")
        refreshPage() if category == "service"
        waitForPageLoad()
    end
end

# Chrome does not wait for objects to load, so some steps need to sleep
# until all the elements load
# For more information:
# http://www.testrisk.com/2015/05/an-error-for-selenium-chrome-vs-firefox.html
def waitForPageLoad()
    if Capybara.default_driver == :selenium_chrome
        sleep 2
    end
end

def refreshPage()
    page.driver.browser.navigate.refresh
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
        return data
    end
end
