Given (/^I have messages$/) do
    @user_page = User.new
    unreadCount = true
    within(@user_page.navbar.userDetails) do
        unreadCount = has_text? 0
    end
    if unreadCount
        visitPoolsPage()
        clickAddPoolButton()
        click_link_or_button("Add Resource Pool")
        fillInResourcePoolField("user page test")
        click_link_or_button("Add Resource Pool")
        clickAddPoolButton()
        fillInResourcePoolField("user page test")
        click_link_or_button("Add Resource Pool")
        closeDialog()
    end
end

When(/^I view user details$/) do
    viewUserDetails()
end

When (/^I clear my messages$/) do
    @user_page.clearMessages_button.click()
end

When (/^I click on the first unread message$/) do
    defaultMatch = Capybara.match
    Capybara.match = :first
    @user_page.unreadMessage.click()
    Capybara.match = defaultMatch
end

When (/^I switch the language to English$/) do
    @user_page.english_button.click()
end

When (/^I switch the language to Spanish$/) do
    @user_page.spanish_button.click()
end

Then /^I should see the name of the current user$/ do
    name = applicationUserID()
    expect(@user_page).to have_content "Username: #{name}"
end

Then /^I should see my messages$/ do
    defaultMatch = Capybara.match
    Capybara.match = :first
    expect(@user_page.message.visible?).to be true
    Capybara.match = defaultMatch
end

Then /^I should not see any messages$/ do
    expect(@user_page.has_selector?("div[class^='message ']")).to be false
end

Then /^I should see that the first unread message is marked as read$/ do
    defaultMatch = Capybara.match
    Capybara.match = :first
    expect(@user_page.message[:class]).to include("message readMessage ng-scope")
    Capybara.match = defaultMatch
end

def viewUserDetails()
    @user_page = User.new
    @user_page.navbar.userDetails.click()
end