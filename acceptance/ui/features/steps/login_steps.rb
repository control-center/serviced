When(/^I am on the login page$/) do
    visitLoginPage()
end

When(/^I fill in the user id field with "(.*?)"$/) do |userid|
    @login_page.userid_field.set userid
end

When(/^I fill in the user id field with the default user id$/) do
    fillInDefaultUserID()
end

When(/^I fill in the password field with "(.*?)"$/) do |password|
    @login_page.password_field.set password
end

When(/^I fill in the password field with the default password$/) do
    fillInDefaultPassword()
end

#
# This is a bit of an anti-pattern because creating highly specific steps for each page
#    when a generic step will suffice creates lots of extra steps, leading to more
#    maintenance overhead. However, in this case the github login page has 2 buttons
#    labeled "Sign in" without ids, so it was impossible to use the more generic
#    when-I-click step.
#
When (/^I click the sign-in button$/) do
    clickSignInButton()
end

And (/^I close the deploy wizard if present$/) do
    closeDeployWizard()
end

def visitLoginPage()
    oldWait = setDefaultWaitTime(180)
    @login_page = Login.new
    @login_page.load
    expect(@login_page).to be_displayed
    setDefaultWaitTime(oldWait)

    # wait till loading animation clears
    @login_page.has_no_css?(".loading_wrapper")
end

def fillInDefaultUserID()
    @login_page.userid_field.set applicationUserID()
end

def fillInDefaultPassword()
    @login_page.password_field.set applicationPassword()
end

def clickSignInButton()
    @login_page.signin_button.click
end

Then(/^I should see the login error "(.*?)"$/) do |text|
    expect(@login_page.error_message.text).to have_content text
end
