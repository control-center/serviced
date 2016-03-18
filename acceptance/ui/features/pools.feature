@pools @screenshot
Feature: Resource Pool Management
  In order to use Control Center
  As a CC admin user
  I want to manage resource pools

  Background:
    Given that the admin user is logged in

  Scenario: View default resource pools page
    When I am on the resource pool page
    Then I should see "Resource Pools"
      And I should see the add Resource Pool button
      And I should see "Memory Usage"
      And I should see "Created"
      And I should see "Last Modified"

  Scenario: View Add Resource Pool dialog
    When I am on the resource pool page
      And I click the add Resource Pool button
    Then I should see "Add Resource Pool"
      And I should see "Resource Pool: "
      And I should see the Resource Pool name field
      And I should see "Description: "
      And I should see the Description field

  Scenario: Add a resource pool with a duplicate name
    Given that the default resource pool is added
    When I am on the resource pool page
      And I click the add Resource Pool button
      And I fill in the Resource Pool name field with "table://pools/defaultPool/name"
      And I fill in the Description field with "table://pools/defaultPool/description"
      And I click "Add Resource Pool"
    Then I should see "Adding pool failed"
      And I should see "Internal Server Error: facade: resource pool exists"

  Scenario: Add a resource pool without specifying a name
    When I am on the resource pool page
      And I click the add Resource Pool button
      And I fill in the Description field with "none"
      And I click "Add Resource Pool"
    Then I should see "Adding pool failed"
      And I should see "Internal Server Error: empty Kind id"

  @clean_pools
  Scenario: Add a resource pool
    Given that only the default resource pool is added
    When I am on the resource pool page
      And I click the add Resource Pool button
      And I fill in the Resource Pool name field with "table://pools/pool2/name"
      And I fill in the Description field with "table://pools/pool2/description"
      And I click "Add Resource Pool"
    Then I should see "Added new Pool"
      And I should see "Added resource pool"
      And I should see an entry for "table://pools/pool2/name" in the table
      And the "table://pools/defaultPool/name" button should be disabled
      And the "table://pools/pool2/name" button should not be disabled

  @clean_pools
  Scenario: Delete a resource pool
    When I am on the resource pool page
      And I add the "pool2" pool
    Then I should see an entry for "table://pools/pool2/name" in the table
      And I remove "table://pools/pool2/name"
    Then I should see "This action will permanently delete the resource pool"
    When I click "Remove Pool"
    Then I should see "Removed Pool"
      And I should not see an entry for "table://pools/pool2/name" in the table

  @clean_hosts
  Scenario: Check resource pool data when hosts are added
    Given only the default host is added
    When I am on the hosts page
      And I add the "host2" host
      And I am on the resource pool page
    Then I should see the sum of "table://hosts/defaultHost/cores, table://hosts/host2/cores" in the "CPU Cores" column
      And I should see the sum of "table://hosts/defaultHost/memoryGB, table://hosts/host2/memoryGB" in the "Memory Usage" column
