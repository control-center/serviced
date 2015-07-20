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
    removeAllEntries("host")
end

After('@clean_pools') do
    visitPoolsPage()
    removeAllPools()
    addDefaultPool() # default pool must exist or else serviced log gets spammed CC-1105
end

After('@clean_templates') do
    visitApplicationsPage()
    removeAllEntries("template")
end

After('@clean_services') do
    visitApplicationsPage()
    removeAllEntries("service")
end