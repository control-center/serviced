@apps_sorting
Feature: Application Sorting
  In order to use Control Center
  As a CC admin user
  I want to sort applications and application templates

  Background:
    Given that the admin user is logged in
      And that multiple applications and application templates have been added

  Scenario: Sort applications by ascending name
    When I am on the applications page
      And I sort by "Application" in ascending order
    Then the "Application" column should be sorted in ascending order

  Scenario: Sort application templates by ascending description
    When I am on the applications page
      And I sort by "Description" in ascending order
    Then the "Description" column should be sorted in ascending order

  @clean_services @clean_templates
  Scenario: Sort application templates by descending description
    When I am on the applications page
      And I sort by "Description" in descending order
    Then the "Description" column should be sorted in descending order
