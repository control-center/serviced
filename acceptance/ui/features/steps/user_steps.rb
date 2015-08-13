Given (/^(?:|that )I have messages$/) do
    @user_page = User.new
    zeroMessages = @user_page.navbar.userDetails.has_text? 0
    if zeroMessages
        visitHostsPage()
        removeAllEntries("host")
        addDefaultHost()
        removeAllEntries("host")
    end
end

When (/^I view user details$/) do
    viewUserDetails()
end

When (/^I clear my messages$/) do
    @user_page.wait_for_clearMessages_button
    @user_page.clearMessages_button.click()
end

When (/^I click on the first unread message$/) do
    defaultMatch = Capybara.match
    Capybara.match = :first
    @user_page.wait_for_unreadMessage
    @user_page.unreadMessage.click()
    Capybara.match = defaultMatch
end

When (/^I switch the language to English$/) do
    @user_page.wait_for_english_button
    @user_page.english_button.click()
end

When (/^I switch the language to Spanish$/) do
    @user_page.wait_for_spanish_button
    @user_page.spanish_button.click()
end

Then (/^I should see the name of the current user$/) do
    name = applicationUserID()
    expect(@user_page).to have_content "Username: #{name}"
end

Then (/^I should see my messages$/) do
    defaultMatch = Capybara.match
    Capybara.match = :first
    expect(@user_page.message.visible?).to be true
    Capybara.match = defaultMatch
end

Then (/^I should not see any messages$/) do
    @user_page.wait_until_message_invisible
    expect(@user_page).to have_no_css("div[class^='message ']")
end

Then (/^I should see that the first unread message is marked as read$/) do
    defaultMatch = Capybara.match
    Capybara.match = :first
    expect(@user_page.message[:class]).to include("message readMessage ng-scope")
    Capybara.match = defaultMatch
end

def viewUserDetails()
    @user_page = User.new
    @user_page.navbar.userDetails.click()
end