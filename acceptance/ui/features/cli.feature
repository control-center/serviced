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

  @port
  Scenario: List the port public endpoints
    Given that the "port0" port is added
    Then I should see the port public endpoint "port0" in the list output

  @port
  Scenario: Add a new port public endpoint
    Given that the "port1" port does not exist
      And that the "port1" port is added
    Then I should see the port public endpoint "port1" in the service

  @port
  Scenario: Delete the port public endpoint
    Given that the "port0" port is added
      And the port public endpoint "port0" is removed
    Then I should not see the port public endpoint "port0" in the service

  @port
  Scenario: Disable and enable a port public endpoint
    Given that the "port0" port is added
      And that the port public endpoint "port0" is disabled
    Then the port public endpoint "port0" should be "disabled" in the service
      And that the port public endpoint "port0" is enabled
    Then the port public endpoint "port0" should be "enabled" in the service

  @vhost
  Scenario: List the vhost public endpoints
    Given that the "vhost0" vhost is added
    Then I should see the vhost public endpoint "vhost0" in the list output

  @vhost
  Scenario: Add a new vhost public endpoint
    Given that the "vhost1" vhost does not exist
      And that the "vhost1" vhost is added
    Then I should see the vhost public endpoint "vhost1" in the service

  @vhost
  Scenario: Delete the vhost public endpoint
    Given that the "vhost0" vhost is added
      And the vhost public endpoint "vhost0" is removed
    Then I should not see the vhost public endpoint "vhost0" in the service

  @vhost
  Scenario: Disable a port vhost endpoint
    Given that the "vhost0" vhost is added
      And that the vhost public endpoint "vhost0" is disabled
    Then the vhost public endpoint "vhost0" should be "disabled" in the service

  @vhost
  Scenario: Disable and enable a vhost public endpoint
    Given that the "vhost0" vhost is added
      And that the vhost public endpoint "vhost0" is disabled
      And that the vhost public endpoint "vhost0" is enabled
    Then the vhost public endpoint "vhost0" should be "enabled" in the service
