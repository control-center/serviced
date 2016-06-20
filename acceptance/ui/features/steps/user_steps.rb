Given (/^(?:|that )I have messages$/) do
    zeroMessages = CC.UI.UserPage.navbar.userDetails.has_text? 0
    if zeroMessages
        visitHostsPage()
        removeAllEntries("host")
        CC.CLI.host.add_default_host()
        removeAllEntries("host")
    end
end

When (/^I view user details$/) do
    viewUserDetails()
end

When (/^I clear my messages$/) do
    CC.UI.UserPage.wait_for_clearMessages_button
    CC.UI.UserPage.clearMessages_button.click()
end

When (/^I click on the first unread message$/) do
    defaultMatch = Capybara.match
    Capybara.match = :first
    CC.UI.UserPage.wait_for_unreadMessage
    CC.UI.UserPage.unreadMessage.click()
    Capybara.match = defaultMatch
end

When (/^I switch the language to English$/) do
    CC.UI.UserPage.wait_for_english_button
    CC.UI.UserPage.english_button.click()
end

When (/^I switch the language to Spanish$/) do
    CC.UI.UserPage.wait_for_spanish_button
    CC.UI.UserPage.spanish_button.click()
end

Then (/^I should see the name of the current user$/) do
    name = applicationUserID()
    expect(CC.UI.UserPage).to have_content "Username: #{name}"
end

Then (/^I should see my messages$/) do
    defaultMatch = Capybara.match
    Capybara.match = :first
    expect(CC.UI.UserPage.message.visible?).to be true
    Capybara.match = defaultMatch
end

Then (/^I should not see any messages$/) do
    CC.UI.UserPage.wait_until_message_invisible
    expect(CC.UI.UserPage).to have_no_css("div[class^='message ']")
end

Then (/^I should see that the first unread message is marked as read$/) do
    defaultMatch = Capybara.match
    Capybara.match = :first
    expect(CC.UI.UserPage.message[:class]).to include("message readMessage ng-scope")
    Capybara.match = defaultMatch
end

def viewUserDetails()
    CC.UI.UserPage.navbar.userDetails.click()
end