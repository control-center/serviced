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

  @login-required
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
