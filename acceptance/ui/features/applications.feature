@apps
Feature: Application Management
  In order to use Control Center
  As a CC admin user
  I want to manage applications

  @login-required
  Scenario: View default applications page
    When I am on the applications page
    Then I should be on the applications page

  @login-required
  Scenario: View deployment wizard
    When I am on the applications page
      And I click the Add-Application button
    Then I should see "Deployment Wizard"
      And I should see "Select the application to install:"
      And I should see "Step 1"

  @login-required
  Scenario: View application template dialog
    When I am on the applications page
      And I click the Add-Application Template button
    Then I should see "Add Application Template"
      And I should see "Application Template File:"

  @login-required
  Scenario: View services map
    When I am on the applications page
      And I click the Services Map button
    Then I should not see "Deployment ID"

  @run @login-required
  Scenario: Sort applications by name
    When I am on the applications page
      And I sort by "Application" in ascending order
    Then the "Application" column should be sorted in ascending order

  @run @login-required
  Scenario: Sort applications by status
    When I am on the applications page
      And I sort by "Status" in descending order
    Then the "Status" column should be sorted in descending order

  @run @login-required
  Scenario: Sort applications by deployment ID
    When I am on the applications page
      And I sort by "Deployment ID" in descending order
    Then the "Deployment ID" column should be sorted in descending order

  @run @login-required
  Scenario: Sort applications templates by name
    When I am on the applications page
      And I sort by "Application Template" in ascending order
    Then the "Application Template" column should be sorted in ascending order

  @run @login-required
  Scenario: Sort application templates by ID
    When I am on the applications page
      And I sort by "ID" in descending order
    Then the "ID" column should be sorted in descending order

  @run @login-required
  Scenario: Sort application templates by description
    When I am on the applications page
      And I sort by "Description" in ascending order
    Then the "Description" column should be sorted in ascending order