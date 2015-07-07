@hosts
Feature: Host Management
  In order to use Control Center
  As a CC admin user
  I want to manage hosts

  Background:
    Given that the admin user is logged in

  Scenario: View empty Hosts page
    Given there are no hosts defined
    When I am on the hosts page
    Then I should see "Hosts Map"
      And I should see "Name"
      And I should see "Active"
      And I should see "Resource Pool"
      And I should see "Memory"
      And I should see "RAM Commitment"
      And I should see an empty Hosts page

  Scenario: View Add Host dialog
    When I am on the hosts page
      And I click the Add-Host button
    Then I should see the Add Host dialog
      And I should see "Host and port"
      And I should see the Host and port field
      And I should see "Resource Pool ID"
      And I should see the Resource Pool ID field
      And I should see "RAM Commitment"
      And I should see the RAM Commitment field

  Scenario: Add an invalid host with an invalid name
    Given there are no hosts defined
    When I am on the hosts page
      And I click the Add-Host button
      And I fill in the Host Name field with "bogushost"
      And I fill in the Resource Pool field with the default resource pool
      And I fill in the RAM Commitment field with the default RAM commitment
      And I click "Add Host"
    Then I should see "Error"
      And I should see "Bad Request"
      And the Host and port field should be flagged as invalid
      And I should see an empty Hosts page

  Scenario: Add an invalid host with an invalid port
    Given there are no hosts defined
    When I am on the hosts page
      And I click the Add-Host button
      And I fill in the Host Name field with "172.17.42.1:9999"
      And I fill in the Resource Pool field with the default resource pool
      And I fill in the RAM Commitment field with the default RAM commitment
      And I click "Add Host"
    Then I should see "Error"
      And I should see "Internal Server Error: dial tcp 172.17.42.1:9999: connection refused"
      And I should see an empty Hosts page

  Scenario: Add an invalid host with an invalid Resource Pool field
    Given there are no hosts defined
    When I am on the hosts page
      And I click the Add-Host button
      And I fill in the Host Name field with the default host name
      And I fill in the RAM Commitment field with the default RAM commitment
      And I click "Add Host"
    Then I should see "Error"
      And I should see "Bad Request: empty poolid not allowed"
      And I should see an empty Hosts page

  Scenario: Add an invalid host with an invalid RAM Commitment field
    Given there are no hosts defined
    When I am on the hosts page
      And I click the Add-Host button
      And I fill in the Host Name field with the default host name
      And I fill in the Resource Pool field with the default resource pool
      And I fill in the RAM Commitment field with "invalidentry"
      And I click "Add Host"
    Then I should see "Error"
      And I should see "Bad Request: Parsing percentage for 'invalidentry'"
      And I should see an empty Hosts page

  Scenario: Fill in the hosts dialog and cancel
    Given there are no hosts defined
    When I am on the hosts page
      And I click the Add-Host button
      And I fill in the Host Name field with the default host name
      And I fill in the Resource Pool field with the default resource pool
      And I fill in the RAM Commitment field with the default RAM commitment
      And I click "Cancel"
    Then I should see an empty Hosts page
      And I should not see "Success"

  @clean_hosts
  Scenario: Add an valid host
    Given there are no hosts defined
    When I am on the hosts page
      And I click the Add-Host button
      And I fill in the Host Name field with the default host name
      And I fill in the Resource Pool field with the default resource pool
      And I fill in the RAM Commitment field with the default RAM commitment
      And I click "Add Host"
    Then I should see "Success"
      And I should see "table://hosts/defaultHost/name" in the "Name" column
      And I should see "default" in the "Resource Pool" column
      And I should see "Showing 1 Result"
  
  @clean_hosts
  Scenario: Add another valid host
    Given only the default host is defined
    When I am on the hosts page
      And I click the Add-Host button
      And I fill in the Host Name field with "table://hosts/host2/nameAndPort"
      And I fill in the Resource Pool field with "table://hosts/host2/pool"
      And I fill in the RAM Commitment field with "table://hosts/host2/commitment"
      And I click "Add Host"
    Then I should see "Success"
      And I should see an entry for "table://hosts/host2/name" in the table
      And I should see "table://hosts/defaultHost/name" in the "Name" column
      And I should see "table://hosts/defaultHost/pool" in the "Resource Pool" column
      And I should see "table://hosts/host2/name" in the "Name" column
      And I should see "table://hosts/host2/pool" in the "Resource Pool" column
      And I should see "Showing 2 Results"

  @clean_hosts
  Scenario: Add a duplicate host
    Given only the default host is defined
    When I am on the hosts page
      And I click the Add-Host button
      And I fill in the Host Name field with the default host name
      And I fill in the Resource Pool field with the default resource pool
      And I fill in the RAM Commitment field with the default RAM commitment
      And I click "Add Host"
    Then I should see "Error"
      And I should see "Internal Server Error: host already exists"

  Scenario: Remove a host
    Given only the default host is defined
    When I am on the hosts page
      And I remove "table://hosts/defaultHost/name"
    Then I should see "This action will permanently delete the host"
    When I click "Remove Host"
    Then I should see "Removed host"
      And I should see an empty Hosts page

  Scenario: View Hosts Map
    When I am on the hosts page
      And I click "Hosts Map"
    Then I should see "By RAM"
      And I should see "By CPU"
      And I should not see "Active"
      