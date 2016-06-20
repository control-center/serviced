When(/^I am on the login page$/) do
    CC.UI.visit_login_page()
end

When(/^I fill in the user id field with "(.*?)"$/) do |userid|
    CC.UI.LoginPage.userid_field.set userid
end

When(/^I fill in the user id field with the default user id$/) do
    CC.UI.LoginPage.userid_field.set applicationUserID()
end

When(/^I fill in the password field with "(.*?)"$/) do |password|
    CC.UI.LoginPage.password_field.set password
end

When(/^I fill in the password field with the default password$/) do
    CC.UI.LoginPage.password_field.set applicationPassword()
end

#
# This is a bit of an anti-pattern because creating highly specific steps for each page
#    when a generic step will suffice creates lots of extra steps, leading to more
#    maintenance overhead. However, in this case the github login page has 2 buttons
#    labeled "Sign in" without ids, so it was impossible to use the more generic
#    when-I-click step.
#
When (/^I click the sign-in button$/) do
    CC.UI.LoginPage.signin_button.click
end

And (/^I close the deploy wizard if present$/) do
    closeDeployWizard()
end

Then(/^I should see the login error "(.*?)"$/) do |text|
    expect(CC.UI.LoginPage.error_message.text).to have_content text
end
