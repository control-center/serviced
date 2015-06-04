@hosts
Feature: Host Management
  In order to use Control Center
  As a CC admin user
  I want to manage hosts

  @login-required
  Scenario: View empty Hosts page
    When I am on the hosts page
    Then I should see "Applications"
      And I should see "Hosts Map"
      And I should see "Name"
      And I should see "Active"
      And I should see "Resource Pool"
      And I should see "Memory"
      And I should see "RAM Commitment"
      And I should see "No Data Found"
      And I should see "Showing 0 Results"

  @login-required
  Scenario: Add a host
    When I am on the hosts page
      And I click the Add-Host button
    Then I should see "Host and port"
      And I should see "Resource Pool ID"
    When I fill in the Host Name field with "gjones-dev:4979"
      And I fill in the Resource Pool field with "default"
      And I fill in the RAM Commitment field with "100%"
      And I click "Add Host"
    Then I should see "gjones-dev"
      And I should see "default"
      And I should see "Delete"
