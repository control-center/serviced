Given (/^I have messages$/) do
  unreadCount = true
  within("button[ng-click='modalUserDetails()']") do
    unreadCount = has_text? 0
  end
  if unreadCount
    visitPoolsPage()
    clickAddPoolButton()
    click_link_or_button("Add Resource Pool")
    fillInResourcePoolField("default")
    click_link_or_button("Add Resource Pool")
    closeDialog()
  end
end

When(/^I view user details$/) do
  viewUserDetails()
end

When (/^I clear my messages$/) do
  page.find("[ng-click='clearMessages()']").click()
end

When (/^I click on the unread message "(.*?)"$/) do |title|
  defaultMatch = Capybara.match
  Capybara.match=:first
  page.find("[class='message unreadMessage ng-scope']", :text => title).click()
  Capybara.match = defaultMatch
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

Then /^I should see that the "(.*?)" message is marked as read$/ do |title|
  defaultMatch = Capybara.match
  Capybara.match=:first
  within("ul[class='well list-group']") do
    message = page.find("[class^='message']", :text => title)
    expect(message[:class]).to include("message readMessage ng-scope")
  end
  Capybara.match = defaultMatch
end

def viewUserDetails()
  within(".navbar-collapse") do
    page.find("[ng-click='modalUserDetails()']").click()
  end
end