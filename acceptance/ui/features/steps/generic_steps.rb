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

When (/^I sort by "([^"]*)"$/) do |category|
    page.find("[class^='header  sortable']", :text => /\A#{category}\z/).click()
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
  Capybara.match=:first
  unsorted = Array.new
  while page.all("[ng-repeat='host in $data']").size != 0 do
    unsorted << page.find("td[data-title-text='#{category}']").text
  end
  sorted = unsorted.sort
  sort == unsort
  Capybara.match=:smart
end

Then (/^the "([^"]*)" column should be sorted in descending order$/) do |category|
  Capybara.match=:first
  unsorted = Array.new
  while page.all("[ng-repeat='host in $data']").size != 0 do
    unsorted << page.find("td[data-title-text='#{category}']").text
  end
  sorted = (unsorted.sort {|x,y| y <=> x})
  sort == unsort
  Capybara.match=:smart
end