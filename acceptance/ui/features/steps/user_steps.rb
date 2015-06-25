When(/^I view user details$/) do
    viewUserDetails()
end

When (/^I clear my messages$/) do
  page.find("[ng-click='clearMessages()']").click()
end

When (/^I click on an unread message$/) do
  Capybara.match=:first
  page.find("[class='message unreadMessage ng-scope']").click()
  Capybara.match=:smart
end

When (/^I switch the language to English$/) do
  page.find("input[value='en_US']").click()
end

When (/^I switch the language to Spanish$/) do
  page.find("input[value='es_US']").click()
end

Then /^I should see my messages$/ do
  page.assert_selector("[ng-repeat='message in messages.messages track by message.id']")
end

Then /^I should not see any messages$/ do
  page.assert_no_selector("[ng-repeat='message in messages.messages track by message.id']")
end

Then /^I should see a checkmark$/ do
  page.assert_selector("[class='message readMessage ng-scope']")
end

def viewUserDetails()
  within(".navbar-collapse") do
    page.find("[ng-click='modalUserDetails()']").click()
  end
end