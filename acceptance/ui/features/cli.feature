@cli
Feature: CLI Validation
  In order to use Control Center
  As a CC admin user
  I want to manage CC using the cli

  Background:
   Given that the default resource pool is added
     And that the test template is added
     And only the default host is added
     And that the "table://applications/defaultApp/template" application with the "table://applications/defaultApp/id" Deployment ID is added

  Scenario: List the port public endpoints
    Given that the "port0" port is added
    When I should see the port public endpoint "port0" in the service

  Scenario: Add a new port public endpoint
    Given that the "port1" port does not exist
      And that the "port1" port is added
    Then I should see the port public endpoint "port1" in the service

  Scenario: Delete the port public endpoint
    Given that the "port0" port is added
      And the port public endpoint "port0" is removed
    Then I should not see the port public endpoint "port0" in the service
