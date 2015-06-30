@apps
Feature: Application Management
  In order to use Control Center
  As a CC admin user
  I want to manage applications

  @login-required
  Scenario: View default applications page
    When I am on the applications page
    Then I should see "Application Template"
      And I should see "Deployment ID"
      And I should see "Virtual Host Names"
      And I should see "Internal Services"
      And I should see "Description"
      And I should see "Services Map"

  @run @login-required
  Scenario: View services map
    When I am on the applications page
      And I click the Services Map button
    Then I should see "Internal Services" in the Services Map

  @login-required
  Scenario: View application template dialog
    When I am on the applications page
      And I click the Add-Application Template button
    Then I should see "Add Application Template"
      And I should see "Application Template File:"
      And I should see "No file chosen"

  @login-required
  Scenario: Add an application
    When I am on the applications page
      And I click the Add-Application button
    Then I should see "Deployment Wizard"
      And I should see "Select the application to install:"
    When I select "Zenoss.core"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "default"
      And I click "Next"
    Then I should see "Zenoss.core has been configured for resource pool default"
    When I fill in the Deployment ID field with "Hbase"
      And I click "Deploy"
    Then I should see "Pulling images"
      And I should see "this may take awhile"
      And I should see "Application deployed"

  @login-required @emptyHostsPage
  Scenario: Add an application when no hosts have been added
    When I am on the applications page
      And I click the Add-Application button
    Then I should see "Deployment Wizard"
      And I should see "Add Host"
    When I fill in the Host and port field with "roei-dev:4979"
      And I fill in the Resource Pool ID field with "default"
      And I fill in the "RAM Commitment" field with "25%"
      And I click "Next"
    Then I should see "Select the application to install:"
    When I select "Zenoss.core"
      And I click "Next"
    Then I should see "Select the resource pool to install to:"
    When I select "default"
      And I click "Next"
    Then I should see "Zenoss.core has been configured for resource pool default"
    When I fill in the Deployment ID field with "HBase"
      And I click "Deploy"
    Then I should see "Pulling images"
      And I should see "this may take awhile"
      And I should see "Application deployed"

  @login-required
  Scenario: Sort applications by ascending name
    When I am on the applications page
      And I sort by "Application" in ascending order
    Then the "Application" column should be sorted in ascending order

  @login-required
  Scenario: Sort applications by descending name
    When I am on the applications page
      And I sort by "Application" in descending order
    Then the "Application" column should be sorted in descending order

  @login-required
  Scenario: Sort applications by descending status
    When I am on the applications page
      And I sort by "Status" in descending order
    Then the "Status" column should be sorted with active applications on the bottom

  @login-required
  Scenario: Sort applications by ascending status
    When I am on the applications page
      And I sort by "Status" in ascending order
    Then the "Status" column should be sorted with active applications on top

  @login-required
  Scenario: Sort applications by ascending deployment ID
    When I am on the applications page
      And I sort by "Deployment ID" in ascending order
    Then the "Deployment ID" column should be sorted in ascending order

  @login-required
  Scenario: Sort applications by descending deployment ID
    When I am on the applications page
      And I sort by "Deployment ID" in descending order
    Then the "Deployment ID" column should be sorted in descending order

  @login-required
  Scenario: Sort applications templates by descending name
    When I am on the applications page
      And I sort by "Application Template" in descending order
    Then the "Application Template" column should be sorted in descending order

  @login-required
  Scenario: Sort applications templates by ascending name
    When I am on the applications page
      And I sort by "Application Template" in ascending order
    Then the "Application Template" column should be sorted in ascending order
  
  @login-required
  Scenario: Sort application templates by ascending ID
    When I am on the applications page
      And I sort by "ID" in ascending order
    Then the "ID" column should be sorted in ascending order

  @login-required
  Scenario: Sort application templates by descending ID
    When I am on the applications page
      And I sort by "ID" in descending order
    Then the "ID" column should be sorted in descending order

  @login-required
  Scenario: Sort application templates by ascending description
    When I am on the applications page
      And I sort by "Description" in ascending order
    Then the "Description" column should be sorted in ascending order

  @login-required
  Scenario: Sort application templates by descending description
    When I am on the applications page
      And I sort by "Description" in descending order
    Then the "Description" column should be sorted in descending order

  @login-required
  Scenario: Remove an Application Template
    When I am on the applications page
      And I remove "Zenoss.core" from the Application Templates list
    Then I should see "Remove Template"
      And I should see "This action will permanently delete the template"
    When I click "Remove Template"
    Then I should see "Removed Template"

  @login-required
  Scenario: Remove an application
    When I am on the applications page
      And I remove "Zenoss.core" from the Applications list
    Then I should see "Remove Application"
      And I should see "This action will permanently delete the running application"
    When I click "Remove Application"
    Then I should see "Removed App"
      And I should not see "HBase"
