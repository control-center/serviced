When(/^I am on the applications page$/) do
    visitApplicationsPage()
end

Then(/^I should be on the applications page$/) do
    @applications_page = Applications.new
    expect(@applications_page).to be_displayed
end

def visitApplicationsPage()
    @applications_page = Applications.new
    within(".navbar-collapse") do
        click_link("Applications")
    end
    expect(@applications_page).to be_displayed
end
