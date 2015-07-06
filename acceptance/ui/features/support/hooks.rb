#
# See https://github.com/cucumber/cucumber/wiki/Hooks for more info about hooks
#
Before('@login-required') do
    visitLoginPage()
    fillInDefaultUserID()
    fillInDefaultPassword()
    clickSignInButton()
end

After('@clean_hosts') do
    visitHostsPage()
    removeAllHosts()
end

After('@clean_pools') do
    visitPoolsPage()
    removeAllAddedPools()
end