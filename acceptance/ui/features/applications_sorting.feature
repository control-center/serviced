@apps_sorting @screenshot
Feature: Application Sorting
  In order to use Control Center
  As a CC admin user
  I want to sort applications and application templates

  Background:
    Given PENDING that multiple applications and application templates have been added
      And that the admin user is logged in

  Scenario: Sort applications by ascending name
    When I am on the applications page
      And I sort by "Application" in ascending order
    Then the "Application" column should be sorted in ascending order
  
  Scenario: Sort applications by descending name
    When I am on the applications page
      And I sort by "Application" in descending order
    Then the "Application" column should be sorted in descending order
  
  Scenario: Sort applications by descending status
    When I am on the applications page
      And I sort by "Status" in descending order
    Then the "Status" column should be sorted with active applications on the bottom

  Scenario: Sort applications by ascending status
    When I am on the applications page
      And I sort by "Status" in ascending order
    Then the "Status" column should be sorted with active applications on top

  Scenario: Sort applications by ascending deployment ID
    When I am on the applications page
      And I sort by "Deployment ID" in ascending order
    Then the "Deployment ID" column should be sorted in ascending order

  Scenario: Sort applications by descending deployment ID
    When I am on the applications page
      And I sort by "Deployment ID" in descending order
    Then the "Deployment ID" column should be sorted in descending order

  Scenario: Sort application templates by descending name
    When I am on the applications page
      And I sort by "Application Template" in descending order
    Then the "Application Template" column should be sorted in descending order

  Scenario: Sort application templates by ascending name
    When I am on the applications page
      And I sort by "Application Template" in ascending order
    Then the "Application Template" column should be sorted in ascending order
  
  Scenario: Sort application templates by ascending ID
    When I am on the applications page
      And I sort by "ID" in ascending order
    Then the "ID" column should be sorted in ascending order

  Scenario: Sort application templates by descending ID
    When I am on the applications page
      And I sort by "ID" in descending order
    Then the "ID" column should be sorted in descending order

  Scenario: Sort application templates by ascending description
    When I am on the applications page
      And I sort by "Description" in ascending order
    Then the "Description" column should be sorted in ascending order

  @clean_services @clean_templates
  Scenario: Sort application templates by descending description
    When I am on the applications page
      And I sort by "Description" in descending order
    Then the "Description" column should be sorted in descending order
