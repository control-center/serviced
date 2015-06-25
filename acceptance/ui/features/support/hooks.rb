#
# See https://github.com/cucumber/cucumber/wiki/Hooks for more info about hooks
#
Before('@login-required') do
  visitLoginPage()
  fillInDefaultUserID()
  fillInDefaultPassword()
  clickSignInButton()
end

Before('@emptyHostsPage') do
  visitHostsPage()
  removeAllHosts()
end

Before ('@defaultHostPage') do
  visitHostsPage()
  removeAllHosts()
  clickAddHostButton()
  fillInDefaultHostAndPort()
  fillInDefaultResourcePool()
  fillInDefaultRAMCommitment()
  click_link_or_button("Add Host")
end