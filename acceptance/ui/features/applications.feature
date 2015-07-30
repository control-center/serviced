@apps
Feature: Application Management
  In order to use Control Center
  As a CC admin user
  I want to manage applications

  Background:
    Given that the admin user is logged in

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
      And I click the Add-Application Template button
    Then I should see "Add Application Template"
      And I should see "Application Template File:"

  @clean_hosts
  Scenario: Deploy Zenoss Core
    Given that Zenoss Core is not added
      And only the default host is defined
    When I am on the applications page
      And I click the Add-Application button
    Then I should see "Deployment Wizard"
      And I should see "Select the application to install:"
    When I select "Zenoss.core"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "default"
      And I click "Next"
    Then I should see "Zenoss.core has been configured for resource pool default."
      And I should see "Deployment ID"
    When I fill in the Deployment ID field with "test"
      And I click "Deploy"
    Then I should see that the application has deployed

  @clean_hosts
  Scenario: Deploy Zenoss Core and add a host
    Given that Zenoss Core is not added
      And there are no hosts defined
    When I am on the applications page
      And I click the Add-Application button
    Then I should see "Deployment Wizard"
      And I should see "Add Host"
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
    When I fill in the Deployment ID field with "test"
      And I click "Deploy"
    Then I should see that the application has deployed