When /^I fill in "([^"]*)" with "([^"]*)"$/ do |element, text|
  fill_in element, with: text
end

When(/^I fill in the "([^"]*)" field with "(.*?)"$/) do |field, text|
  find(field).set(text)
end

When /^I click "([^"]*)"$/ do |text|
  click_link_or_button(text)
end

Then(/^I should see "(.*?)"$/) do |text|
  expect(page).to have_content text
end
