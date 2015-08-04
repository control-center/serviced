@apps
Feature: Application Management
  In order to use Control Center
  As a CC admin user
  I want to manage applications

  Background:
    Given that the admin user is logged in
      And that the default resource pool is added

  Scenario: View default applications page
    When I am on the applications page
    Then I should see "Application Template"
      And I should see "Deployment ID"
      And I should see "Virtual Host Names"
      And I should see "Internal Services"
      And I should see "Description"
      And I should see "Services Map"

  Scenario: View services map
    When I am on the applications page
      And I click the Services Map button
    Then I should see "Internal Services" in the Services Map

  Scenario: View application template dialog
    When I am on the applications page
      And I click the add Application Template button
    Then I should see "Add Application Template"
      And I should see "Application Template File:"

  @clean_hosts
  Scenario: Enter invalid input into the application deployment wizard
    Given that Zenoss Core is not added
      And there are no hosts added
    When I am on the applications page
      And I click the add Application button
    Then I should see "Add Host"
    When I click "Next"
    Then I should see "Please enter a valid host name"
    When I fill in the Host Name field with "bogushost"
      And I fill in the Resource Pool field with "default"
      And I fill in the RAM Commitment field with "100%"
      And I click "Next"
    Then I should see "Add Host failed"
    When I fill in the Host Name field with "table://hosts/defaultHost/nameAndPort"
      And I click "Next"
      And I select "Zenoss.core"
      And I click "Next"
      And I select "default"
      And I click "Next"
      And I click "Deploy"
    Then I should see "You must provide a Deployment ID."
    When I close the dialog
    Then I should not see "Deployment Wizard"

  @clean_hosts @clean_services
  Scenario: Deploy Zenoss Core
    Given that Zenoss Core is not added
      And only the default host is added
    When I am on the applications page
      And I click the add Application button
    Then I should see "Deployment Wizard"
      And I should see "Select the application to install:"
    When I select "Zenoss.core"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "default"
      And I click "Next"
    Then I should see "Zenoss.core has been configured for resource pool default."
      And I should see "Deployment ID"
    When I fill in the Deployment ID field with "table://applications/defaultCore/id"
      And I click "Deploy"
    Then I should see "Pulling image"
      And I should see that the application has deployed
      And I should see an entry for "Zenoss.core" in the Applications table
      And I should see "Showing 2 Results"

  @clean_hosts @clean_services
  Scenario: Deploy Zenoss Core and add a host
    Given that Zenoss Core is not added
      And there are no hosts added
    When I am on the applications page
      And I click the add Application button
    Then I should see "Deployment Wizard"
    When I fill in the Host Name field with "table://hosts/defaultHost/nameAndPort"
      And I fill in the Resource Pool field with "table://hosts/defaultHost/pool"
      And I fill in the RAM Commitment field with "table://hosts/defaultHost/commitment"
      And I click "Next"
    Then I should see "Select the application to install:"
    When I select "Zenoss.core"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "default"
      And I click "Next"
    Then I should see "Zenoss.core has been configured for resource pool default."
      And I should see "Deployment ID"
    When I fill in the Deployment ID field with "table://applications/defaultCore/id"
      And I click "Deploy"
    Then I should see "Pulling image"
      And I should see that the application has deployed
      And I should see an entry for "Zenoss.core" in the Applications table
      And I should see "Showing 2 Results"

  @clean_hosts @clean_services @clean_pools
  Scenario: Deploy Zenoss Core to another resource pool
    Given only the default host is added
      And that Zenoss Core is not added
      And that the "table://applications/testCore/pool" pool is added
    When I am on the applications page
      And I click the add Application button
    Then I should see "Deployment Wizard"
      And I should see "Select the application to install:"
    When I select "Zenoss.core"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "table://applications/testCore/pool"
      And I click "Next"
    Then I should see "Zenoss.core has been configured"
      And I should see "Deployment ID"
    When I fill in the Deployment ID field with "table://applications/testCore/id"
      And I click "Deploy"
    Then I should see "Pulling image"
      And I should see that the application has deployed
      And I should see an entry for "Zenoss.core" in the Applications table
      And I should see "Showing 2 Results"

  @clean_hosts @clean_services
  Scenario: Add Zenoss Core with a duplicate Deployment ID
    Given only the default host is added
      And that Zenoss Core with the "table://applications/defaultCore/id" Deployment ID is added
    When I am on the applications page
      And I click the add Application button
    Then I should see "Deployment Wizard"
      And I should see "Select the application to install:"
    When I select "Zenoss.core"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "default"
      And I click "Next"
    Then I should see "Zenoss.core has been configured for resource pool default."
      And I should see "Deployment ID"
    When I fill in the Deployment ID field with "table://applications/defaultCore/id"
      And I click "Deploy"
    Then I should see that the application has not been deployed
      And I should see "Internal Server Error: deployment ID"
      And I should see "is already in use"
      And I should see "Showing 2 Results"

  @clean_hosts @clean_services
  Scenario: Add Zenoss Core with another Deployment ID
    Given only the default host is added
      And that Zenoss Core with the "table://applications/defaultCore/id" Deployment ID is added
    When I am on the applications page
      And I click the add Application button
    Then I should see "Deployment Wizard"
      And I should see "Select the application to install:"
    When I select "Zenoss.core"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "default"
      And I click "Next"
    Then I should see "Zenoss.core has been configured for resource pool default."
      And I should see "Deployment ID"
    When I fill in the Deployment ID field with "table://applications/testCore/id"
      And I click "Deploy"
    Then I should see "Pulling image"
      And I should see that the application has deployed
      And I should see an entry for "Zenoss.core" in the Applications table
      And I should see "Showing 3 Results"

  @clean_hosts
  Scenario: Remove Zenoss Core
    Given only the default host is added
      And that Zenoss Core with the "table://applications/defaultCore/id" Deployment ID is added
    When I remove "Zenoss.core" from the Applications list
    Then I should see "Remove Application"
      And I should see "This action will permanently delete the running application"
    When I click "Remove Application"
    Then I should see "Removed App"
      And I should not see "Showing 2 Results"
