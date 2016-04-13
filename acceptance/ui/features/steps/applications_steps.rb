# FIXME: This step is a lie; need to fix this to actually deploy more than 1 application
# Otherwise, the application_sorting tests are sorting a list of either 0 or 1 items
# (depending on previous tests)
Given (/^(?:|that )multiple applications and application templates have been added$/) do
    visitApplicationsPage()
    within(@applications_page.services_table) do
        if has_text?("Showing 1 Result")
            # add application
        end
    end
    within(@applications_page.templates_table) do
        if has_text?("Showing 0 Results") || has_text?("Showing 1 Result")
            # add application templates
        end
    end
end

Given (/^(?:|that )the "(.*?)" application is not added$/) do |app|
    exists = checkServiceExistsCLI(app)
    removeServiceCLI(app) if exists
end

Given (/^(?:|that )the "(.*?)" application with the "(.*?)" Deployment ID is added$/) do |app, id|
    visitApplicationsPage()
    exists = checkServiceRows(app) && isInColumn(id, "Deployment ID")
    addService(app, "default", id) if !exists
end

# Note this step definition is optimized to use the CLI exclusively so that it can be called before user login
Given (/^(?:|that )the test template is added$/) do
    exists = checkTemplateExistsCLI("testsvc")
    addTemplateCLI(TEMPLATE_DIR) if !exists
end

When(/^I am on the applications page for the first time$/) do
    # Try to visit the application page.  Retry twice, with 10 second delays.
    retryMethod(method(:visitApplicationsPage), 2, 10)
end

When(/^I am on the applications page$/) do
    visitApplicationsPage()
end

When(/^I click the add Application button$/) do
    @applications_page.addApp_button.click()
end

When(/^I click the add Application Template button$/) do
    @applications_page.addAppTemplate_button.click()
end

When(/^I click the Services Map button$/) do
    @applications_page.servicesMap_button.click()
    @servicesMap_page = ServicesMap.new
end

When(/^I fill in the Deployment ID field with "(.*?)"$/) do |deploymentID|
    fillInDeploymentID(deploymentID)
end

When(/^I remove "(.*?)" from the Applications list$/) do |name|
    within(@applications_page.services_table, :text => name) do
        click_link_or_button("Delete")
    end
end

When(/^I remove "(.*?)" from the Application Templates list$/) do |name|
    within(@applications_page.templates_table, :text => name) do
        click_link_or_button("Delete")
    end
end

Then (/^I should see that the application has deployed$/) do
    expect(page).to have_content("App deployed successfully", wait: 120)
    #refreshPage() # workaround until apps consistently display on page without refreshing
end

Then (/^I should see that the application has not been deployed$/) do
    expect(page).to have_content("App deploy failed")
end

Then (/^the "Status" column should be sorted with active applications on (top|the bottom)$/) do |order|
    list = @applications_page.status_icons
    for i in 0..(list.size - 2)
        if order == "top"
            # assuming - (ng-isolate-scope down) before + (ng-isolate-scope good)
            expect(list[i][:class]).to be <= list[i + 1][:class]
        else
            expect(list[i][:class]).to be >= list[i + 1][:class]    # assuming + before - before !
        end
    end
end

Then (/^I should see "(.*?)" in the Services Map$/) do |node|
    within(@servicesMap_page.map) do
        assert_text(getTableValue(node))
    end
end

Then (/^I should see an entry for "(.*?)" in the Applications table$/) do |entry|
    expect(checkServiceRows(entry)).to be true
end

Then (/^I should see an entry for "(.*?)" in the Application Templates table$/) do |entry|
    expect(checkTemplateRows(entry)).to be true
end

Then (/^"(.*?)" should be active$/) do |entry|
    expect(checkActive(entry)).to be true
end


def checkActive(entry)
    within(page.find("table[data-config='servicesTable']")) do
        within(page.find("tr", :text => entry)) do
            return page.has_css?("[class*='passed']")
        end
    end
end

def checkServiceRows(row)
    found = false
    within(@applications_page.services_table) do
        found = page.has_text?(getTableValue(row))
    end
    return found
end

def checkTemplateRows(row)
    found = false
    within(@applications_page.templates_table) do
        found = page.has_text?(getTableValue(row))
    end
    return found
end

def visitApplicationsPage()
    @applications_page = Applications.new
    @applications_page.load
    expect(@applications_page).to be_displayed
    closeDeployWizard()
end

def fillInDeploymentID(id)
    @applications_page.deploymentID_field.set getTableValue(id)
end

def addService(name, pool, id)
    @applications_page.addApp_button.click()
    selectOption(name)
    click_link_or_button("Next")
    selectOption(pool)
    click_link_or_button("Next")
    fillInDeploymentID(id)
    click_link_or_button("Deploy")
    expect(page).to have_content("App deployed successfully", wait: 120)
end

def checkServiceExistsCLI(serviceName)
    serviceName = getTableValue(serviceName)
    servicedCLI = getServicedCLI()

    result = `#{servicedCLI} service list --show-fields Name 2>&1`
    verifyCLIExitSuccess($?, result)

    matchData = result.match /^#{serviceName}$/
    return matchData != nil
end

def removeServiceCLI(serviceName)
    serviceName = getTableValue(serviceName)
    servicedCLI = getServicedCLI()

    result = `#{servicedCLI} service rm #{serviceName} 2>&1`
    verifyCLIExitSuccess($?, result)
end

def removeAllServicesCLI()
    servicedCLI = getServicedCLI()

    result = `#{servicedCLI} service list --show-fields ServiceID 2>&1 | grep -v ServiceID | xargs --no-run-if-empty #{servicedCLI} service rm 2>&1`
    verifyCLIExitSuccess($?, result)
end

def addTemplateCLI(dir)
    servicedCLI = getServicedCLI()
    result = `#{servicedCLI} template compile #{dir} | #{servicedCLI} template add`
    verifyCLIExitSuccess($?, result)

    templateID = result
    result = `#{servicedCLI} template list #{templateID}`

    verifyCLIExitSuccess($?, result)
    expect(result.lines.count).not_to eq(0)
end

def checkTemplateExistsCLI(templateName)
    templateName = getTableValue(templateName)
    servicedCLI = getServicedCLI()
    result = `#{servicedCLI} template list --show-fields Name 2>&1`
    verifyCLIExitSuccess($?, result)

    matchData = result.match /^#{templateName}$/
    return matchData != nil
end

def removeAllTemplatesCLI()
    servicedCLI = getServicedCLI()

    result = `#{servicedCLI} template list --show-fields TemplateID 2>&1 | grep -v TemplateID | xargs --no-run-if-empty #{servicedCLI} template rm 2>&1`
    verifyCLIExitSuccess($?, result)
end

def closeDeployWizard()
    # if the deploy wizard is on the page and visible, close it
    page_found = false
    begin
        el = find("#addApp")
        # found it!
        if el.visible?
            page_found = true
            el.find(".modal-header .close").click()
        end
    rescue
        # couldn't find the deploy wizard,
        # but that's ok. we all make mistakes
        return
    end

    if page_found
        # wait till it is no longer visible, and error if it remains on screen
        expect(page).not_to have_selector("#addApp")
        # must wait for the modal shadowbox to clear as well
        expect(page).not_to have_selector(".modal-backdrop")
    end
end

