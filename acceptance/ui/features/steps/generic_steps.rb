When (/^I fill in "([^"]*)" with "([^"]*)"$/) do |element, text|
  fill_in element, with: text
end

When (/^I fill in the "([^"]*)" field with "(.*?)"$/) do |field, text|
  find(field).set(text)
end

When (/^I click "([^"]*)"$/) do |text|
  click_link_or_button(text)
end

When (/^I remove "([^"]*)"$/) do |name|
  within("tr[class='ng-scope']", :text => name) do
    click_link_or_button("Delete")
  end
end

When(/^I select "(.*?)"$/) do |name|
  within("tr[class='clickable ng-scope']", :text => name) do
    page.find("input[type='radio']").click()
  end
end

When (/^I sort by "([^"]*)" in ([^"]*) order$/) do |category, sortOrder|
  # class for sortable column headers
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

Then /^I should see "(.*?)"$/ do |text|
  expect(page).to have_content text
end

Then /^I should not see "(.*?)"$/ do |text|
  expect(page).to have_no_content text
end

Then (/^I should see the "([^"]*)"$/) do |element|
  find(element).visible?
end

Then (/^I should see "(.*?)" in the "([^"]*)" column$/) do |text, column|
  # attribute that includes name of column of all table cells
  list = page.all("td[data-title-text='#{column}']")
  list.include?(:text => text)
end

Then (/^the "([^"]*)" column should be sorted in ([^"]*) order$/) do |category, order|
  if order == "ascending"
    sortColumn(category, true)
  else
    sortColumn(category, false)
  end
end


def sortColumn(category, order)
  list = page.all("td[data-title-text='#{category}']")
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