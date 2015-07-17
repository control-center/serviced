Given (/^that the admin user is logged in$/) do
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

When (/^I click "([^"]*)"$/) do |text|
    click_link_or_button(getTableValue(text))
end

When /^I close the dialog$/ do
    closeDialog()
end

When (/^I remove "([^"]*)"$/) do |name|
    entry = getTableValue(name)
    within("tr[class='ng-scope']", :text => entry) do
        click_link_or_button("Delete")
    end
end

When(/^I select "(.*?)"$/) do |name|
    entry = getTableValue(name)
    within("tr[class='clickable ng-scope']", :text => entry) do
        page.find("input[type='radio']").click()
    end
end

When (/^I sort by "([^"]*)" in ([^"]*) order$/) do |category, order|
    sortColumn(category, order)
end

Then /^I should see "(.*?)"$/ do |text|
    expect(page).to have_content getTableValue(text)
end

Then /^I should not see "(.*?)"$/ do |text|
    expect(page).to have_no_content getTableValue(text)
end

Then (/^I should see the "([^"]*)"$/) do |element|
    find(element).visible?
end

Then (/^I should see "(.*?)" in the "([^"]*)" column$/) do |text, column|
    checkColumn(text, column, true)
end

Then (/^I should not see "(.*?)" in the "([^"]*)" column$/) do |text, column|
    checkColumn(text, column, false)
end

Then (/^the "([^"]*)" column should be sorted in ([^"]*) order$/) do |category, order|
    if order == "ascending"
        assertSortedColumn(category, true)
    else
        assertSortedColumn(category, false)
    end
end

Then (/^I should see an entry for "(.*?)" in the table$/) do |row|
    checkRows(row, true)
end

Then (/^I should not see an entry for "(.*?)" in the table$/) do |row|
    checkRows(row, false)
end


def assertSortedColumn(category, order)
    list = page.all("td[data-title-text='#{category}'][sortable]")
    for i in 0..(list.size - 2)
        if category == "Created" || category == "Last Modified"
            if order
                DateTime.parse(list[i].text).should <= DateTime.parse(list[i + 1].text)
            else
                DateTime.parse(list[i].text).should >= DateTime.parse(list[i + 1].text)
            end
        else
            if order
            # Category sorting ignores case
                list[i].text.downcase.should <= list[i + 1].text.downcase
            else
                list[i].text.downcase.should >= list[i + 1].text.downcase
            end
        end
    end
end

def checkRows(row, present)
    found = false
    name = getTableValue(row)
    entries = page.all("tr[ng-repeat$='in $data']")
    for i in 0..(entries.size - 1)
        within(entries[i]) do
            found = true if has_text?(name)
        end
    end
    found.should == present
end

def checkColumn(text, column, present)
    # attribute that includes name of column of all table cells
    list = page.all("td[data-title-text='#{column}']")
    cell = getTableValue(text)
    hasEntry = false
    for i in 0..(list.size - 1)
        hasEntry = true if list[i].text == cell.to_s
    end
    hasEntry.should == present
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

def removeAllEntries()
    entries = page.all("[ng-repeat$='in $data']")
    for i in 0..(entries.size - 1)
        within(entries[i]) do
            click_link_or_button("Delete")
        end
        click_link_or_button("Remove")
    end
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
    elsif propertyName == "nameAndPort"
        return HOST_IP + ":" + PARSED_DATA[tableType][tableName]["rpcPort"].to_s
    else
        return PARSED_DATA[tableType][tableName][propertyName]
    end
end
