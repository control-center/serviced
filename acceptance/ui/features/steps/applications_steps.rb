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

When(/^I fill in the Deployment ID field with "(.*?)"$/) do |deploymentID|
  @applications_page.deploymentID_field.set deploymentID
end

When(/^I remove "(.*?)" from the Applications list$/) do |name|
  within("table[data-config='servicesTable']", :text => name) do
    click_link_or_button("Delete")
  end
end

When(/^I remove "(.*?)" from the Application Templates list$/) do |name|
  within("table[data-config='templatesTable']", :text => name) do
    click_link_or_button("Delete")
  end
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

Then (/^I should see "([^"]*)" in the Services Map$/) do |node|
  within("svg") do
    assert_text(node)
  end
end

def visitApplicationsPage()
    @applications_page = Applications.new
    within(".navbar-collapse") do
        click_link("Applications")
    end
    expect(@applications_page).to be_displayed
end
