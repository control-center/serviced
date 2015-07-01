#
# See https://github.com/cucumber/cucumber/wiki/Hooks for more info about hooks
#
Before('@login-required') do
  visitLoginPage()
  fillInDefaultUserID()
  fillInDefaultPassword()
  clickSignInButton()
end