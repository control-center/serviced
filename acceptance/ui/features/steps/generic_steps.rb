#  rubular.com apparently the ^ and ? regexes are equivalent

When (/^I fill in "([^"]*)" with "([^"]*)"$/) do |element, text|
  fill_in element, with: text
end

When (/^I fill in the "([^"]*)" field with "(.*?)"$/) do |field, text|
  find(field).set(text)
end

When (/^I click "([^"]*)"$/) do |text|
  click_link_or_button(text)
end

When (/^I sort by "([^"]*)" in ascending order$/) do |category|
  categoryLink = page.find("[class^='header  sortable']", :text => /\A#{category}\z/)
  while categoryLink[:class] != 'header  sortable sort-asc' do
    categoryLink.click()
  end
end

When (/^I sort by "([^"]*)" in descending order$/) do |category|
  categoryLink = page.find("[class^='header  sortable']", :text => /\A#{category}\z/)
  while categoryLink[:class] != 'header  sortable sort-desc' do
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

Then (/^I should see "(.*?)" in the "([^"]*)" column$/) do |text, category|
  Capybara.match=:prefer_exact
  within("td[data-title-text='#{category}']") do
    page.assert_text(text);
  end
end

Then (/^the "([^"]*)" column should be sorted in ascending order$/) do |category|
  sortColumn(category, true)
end

Then (/^the "([^"]*)" column should be sorted in descending order$/) do |category|
  sortColumn(category, false)
end


def sortColumn(category, order)
  list = page.all("td[data-title-text='#{category}']")
  for i in 0..(list.size - 1)
    puts list[i].text
  end
  for i in 0..(list.size - 2)
    if order
      list[i].text.should <= list[i+1].text
    else
      list[i].text.should >= list[i+1].text
    end
  end
end