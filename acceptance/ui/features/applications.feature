@apps @screenshot
Feature: Application Management
  In order to use Control Center
  As a CC admin user
  I want to manage applications

  Background:
    Given that the test template is added
      And that the default resource pool is added
      And that the admin user is logged in

  Scenario: View default applications page
    When I am on the applications page for the first time
    Then I should see "Application Template"
      And I should see "Deployment ID"
      And I should see "Public Endpoints"
      And I should see "Internal Services"
      And I should see "Description"
      And I should see "Services Map"

  Scenario: View services map
    Given only the default host is added
      And that the "table://applications/defaultApp/template" application with the "table://applications/defaultApp/id" Deployment ID is added
    When I am on the applications page
      And I click the Services Map button
    Then I should see "table://applications/defaultApp/template" in the Services Map

  Scenario: View application template dialog
    When I am on the applications page
      And I click the add Application Template button
    Then I should see "Add Application Template"
      And I should see "Application Template File:"

  @clean_hosts
  Scenario: Enter invalid input into the application deployment wizard
    Given that the "table://applications/defaultApp/template" application is not added
      And there are no hosts added
    When I am on the applications page
      And I click the add Application button
    Then I should see "Add Host"
    When I click "Next"
    Then I should see "Please enter a valid host name"
    When I fill in the Host field with "bogushost"
      And I fill in the Resource Pool field with "default"
      And I fill in the RAM Limit field with "100%"
      And I click "Next"
    Then I should see "Add Host failed"
    When I fill in the Host field with "table://hosts/defaultHost/hostName"
      And I fill in the Port field with "table://hosts/defaultHost/rpcPort"
      And I click "Next"
      And I select "table://applications/defaultApp/template"
      And I click "Next"
      And I select "table://applications/defaultApp/pool"
      And I click "Next"
      And I click "Deploy"
    Then I should see "You must provide a Deployment ID."
    When I close the dialog
    Then I should not see "Deployment Wizard"

  @clean_hosts @clean_services
  Scenario: Deploy an instance of the default template
    Given that the "table://applications/defaultApp/template" application is not added
      And only the default host is added
    When I am on the applications page
      And I click the add Application button
    Then I should see "Deployment Wizard"
      And I should see "Select the application to install:"
    When I select "table://applications/defaultApp/template"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "table://applications/defaultApp/pool"
      And I click "Next"
    Then I should see "table://applications/defaultApp/template"
      And I should see "has been configured"
      And I should see "Deployment ID"
    When I fill in the Deployment ID field with "table://applications/defaultApp/id"
      And I click "Deploy"
    Then I should see that the application has deployed
      And I should see an entry for "table://applications/defaultApp/template" in the Applications table
      And I should see "Showing 2 Results" in the "Applications" table

  @clean_hosts @clean_services
  Scenario: Deploy an instance of the default template and add a host
    Given that the "table://applications/defaultApp/template" application is not added
      And there are no hosts added
    When I am on the applications page
      And I click the add Application button
    Then I should see "Deployment Wizard"
    When I fill in the Host field with "table://hosts/defaultHost/hostName"
      And I fill in the Port field with "table://hosts/defaultHost/rpcPort"
      And I fill in the Resource Pool field with "table://hosts/defaultHost/pool"
      And I fill in the RAM Limit field with "table://hosts/defaultHost/commitment"
      And I click "Next"
    Then I should see "Select the application to install:"
    When I select "table://applications/defaultApp/template"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "table://applications/defaultApp/pool"
      And I click "Next"
    Then I should see "table://applications/defaultApp/template"
      And I should see "has been configured for resource pool"
      And I should see "table://applications/defaultApp/pool"
      And I should see "Deployment ID"
    When I fill in the Deployment ID field with "table://applications/defaultApp/id"
      And I click "Deploy"
    Then I should see that the application has deployed
      And I should see an entry for "table://applications/defaultApp/template" in the Applications table
      And I should see "Showing 2 Results" in the "Applications" table

  @clean_pools @clean_hosts @clean_services
  Scenario: Deploy an instance of the default template to another resource pool
    Given only the default host is added
      And that the "table://applications/app2/template" application is not added
      And that the "table://applications/app2/pool" pool is added
    When I am on the applications page
      And I click the add Application button
    Then I should see "Deployment Wizard"
      And I should see "Select the application to install:"
    When I select "table://applications/app2/template"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "table://applications/app2/pool"
      And I click "Next"
    Then I should see "table://applications/app2/template"
      And I should see "has been configured"
      And I should see "Deployment ID"
    When I fill in the Deployment ID field with "table://applications/app2/id"
      And I click "Deploy"
    Then I should see that the application has deployed
      And I should see an entry for "table://applications/app2/template" in the Applications table
      And I should see "Showing 2 Results" in the "Applications" table

  @clean_hosts @clean_services
  Scenario: Add an instance of the default template with a duplicate Deployment ID
    Given only the default host is added
      And that the "table://applications/defaultApp/template" application with the "table://applications/defaultApp/id" Deployment ID is added
    When I am on the applications page
      And I click the add Application button
    Then I should see "Deployment Wizard"
      And I should see "Select the application to install:"
    When I select "table://applications/defaultApp/template"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "table://applications/defaultApp/pool"
      And I click "Next"
    Then I should see "table://applications/defaultApp/template"
      And I should see "has been configured for resource pool"
      And I should see "Deployment ID"
    When I fill in the Deployment ID field with "table://applications/defaultApp/id"
      And I click "Deploy"
    Then I should see that the application has not been deployed
      And I should see "Internal Server Error: deployment ID"
      And I should see "is already in use"
      And I should see "Showing 2 Results" in the "Applications" table

  @clean_hosts @clean_services
  Scenario: Add an instance of the default template with another Deployment ID
    Given only the default host is added
      And that the "table://applications/defaultApp/template" application with the "table://applications/defaultApp/id" Deployment ID is added
    When I am on the applications page
      And I click the add Application button
    Then I should see "Deployment Wizard"
      And I should see "Select the application to install:"
    When I select "table://applications/app3/template"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "table://applications/app3/pool"
      And I click "Next"
    Then I should see "table://applications/app3/template"
      And I should see "has been configured for resource pool"
      And I should see "Deployment ID"
    When I fill in the Deployment ID field with "table://applications/app3/id"
      And I click "Deploy"
    Then I should see that the application has deployed
      And I should see an entry for "table://applications/app3/template" in the Applications table
      And I should see "Showing 3 Results" in the "Applications" table

  @clean_hosts
  Scenario: Remove an instance of the default template
    Given only the default host is added
      And that the "table://applications/defaultApp/template" application with the "table://applications/defaultApp/id" Deployment ID is added
    When I remove "table://applications/defaultApp/template" from the Applications list
    Then I should see "Remove Application"
      And I should see "This action will permanently delete the running application"
    When I click "Remove Application"
    Then I should see "Removed App" after waiting no more than "30" seconds
      And I should not see "Showing 2 Results" in the "Applications" table
