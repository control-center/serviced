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
