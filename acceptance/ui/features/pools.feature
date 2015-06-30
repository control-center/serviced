@pools
Feature: Resource Pool Management
  In order to use Control Center
  As a CC admin user
  I want to manage resource pools

  @login-required
  Scenario: View default resource pools page
    When I am on the resource pool page
    Then I should see "Resource Pools"
      And I should see the add Resource Pool button
      And I should see "Memory Usage"
      And I should see "Created"
      And I should see "Last Modified"

  @login-required
  Scenario: View Add Resource Pool dialog
    When I am on the resource pool page
      And I click the add Resource Pool button
    Then I should see "Add Resource Pool"
      And I should see "Resource Pool: "
      And I should see the Resource Pool name field
      And I should see "Description: "
      And I should see the Description field

  @login-required
  Scenario: Add a resource pool with a duplicate name
    When I am on the resource pool page
      And I click the add Resource Pool button
      And I fill in the Resource Pool name field with "default"
      And I fill in the Description field with "none"
      And I click "Add Resource Pool"
    Then I should see "Adding pool failed"
      And I should see "Internal Server Error: facade: resource pool exists"

  @login-required
  Scenario: Add a resource pool without specifying a name
    When I am on the resource pool page
      And I click the add Resource Pool button
      And I fill in the Resource Pool name field with ""
      And I fill in the Description field with "none"
      And I click "Add Resource Pool"
    Then I should see "Adding pool failed"
      And I should see "Internal Server Error: empty Kind id"

  @login-required
  Scenario: Add a resource pool
    When I am on the resource pool page
      And I click the add Resource Pool button
      And I fill in the Resource Pool name field with "test"
      And I fill in the Description field with "test"
      And I click "Add Resource Pool"
    Then I should see "Added new Pool"
      And I should see "Added resource pool"
      And I should see an entry for "test" in the table

  @login-required
  Scenario: Delete a resource pool
    When I am on the resource pool page
      And I remove "test"
    Then I should see "This action will permanently delete the resource pool"
    When I click "Remove Pool"
    Then I should see "Removed Pool"
      And I should not see an entry for "test" in the table

  @login-required
  Scenario: Sort resource pools by ascending name
    When I am on the resource pool page
      And I sort by "Resource Pool" in ascending order
    Then the "Resource Pool" column should be sorted in ascending order

  @login-required
  Scenario: Sort resource pools by descending name
    When I am on the resource pool page
      And I sort by "Resource Pool" in descending order
    Then the "Resource Pool" column should be sorted in descending order

  @login-required
  Scenario: Sort resource pools by ascending number of CPU cores
    When I am on the resource pool page
      And I sort by "CPU Cores" in ascending order
    Then the "CPU Cores" column should be sorted in ascending order

  @login-required
  Scenario: Sort resource pools by descending number of CPU cores
    When I am on the resource pool page
      And I sort by "CPU Cores" in descending order
    Then the "CPU Cores" column should be sorted in descending order

  @login-required
  Scenario: Sort resource pools by descending memory usage
    When I am on the resource pool page
      And I sort by "Memory Usage" in descending order
    Then the "Memory Usage" column should be sorted in descending order

  @login-required
  Scenario: Sort resource pools by ascending memory usage
    When I am on the resource pool page
      And I sort by "Memory Usage" in ascending order
    Then the "Memory Usage" column should be sorted in ascending order

  @login-required
  Scenario: Sort resource pools by ascending creation time
    When I am on the resource pool page
      And I sort by "Created" in ascending order
    Then the "Created" column should be sorted in ascending order

  @login-required
  Scenario: Sort resource pools by descending creation time
    When I am on the resource pool page
      And I sort by "Created" in descending order
    Then the "Created" column should be sorted in descending order

  @login-required
  Scenario: Sort resource pools by descending modification time
    When I am on the resource pool page
      And I sort by "Last Modified" in descending order
    Then the "Last Modified" column should be sorted in descending order

  @login-required
  Scenario: Sort resource pools by ascending modification time
    When I am on the resource pool page
      And I sort by "Last Modified" in ascending order
    Then the "Last Modified" column should be sorted in ascending order
