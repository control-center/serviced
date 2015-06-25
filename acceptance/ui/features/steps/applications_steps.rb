When(/^I am on the applications page$/) do
    visitApplicationsPage()
end

When(/^I click the Add-Application button$/) do
  @applications_page.addApp_button.click()
end

When(/^I click the Add-Application Template button$/) do
  @applications_page.addAppTemplate_button.click()
end

When(/^I click the Services Map button$/) do
  @applications_page.servicesMap_button.click()
end

Then(/^I should be on the applications page$/) do
    @applications_page = Applications.new
    expect(@applications_page).to be_displayed
end

Then /^the "Status" column should be sorted with active applications on (top|the bottom)$/ do |order|
  list = page.all("[data-status$='service.status']")
  for i in 0..(list.size - 2)
    if order == "top"
       # assuming - (ng-isolate-scope down) before + (ng-isolate-scope good)
      list[i][:class] <= list[i + 1][:class]
    else
      list[i][:class] >= list[i + 1][:class]    # assuming + before - before !
    end
  end
end

def visitApplicationsPage()
    @applications_page = Applications.new
    within(".navbar-collapse") do
        click_link("Applications")
    end
    expect(@applications_page).to be_displayed
end
