@pools_sorting @screenshot
Feature: Resource Pool Sorting
  In order to use Control Center
  As a CC admin user
  I want to sort resource pools according to various attributes

  Background:
    Given that the admin user is logged in
      And that multiple resource pools have been added

  Scenario: Sort resource pools by ascending name
    When I am on the resource pool page
      And I sort by "Resource Pool" in ascending order
    Then the "Resource Pool" column should be sorted in ascending order

  Scenario: Sort resource pools by descending name
    When I am on the resource pool page
      And I sort by "Resource Pool" in descending order
    Then the "Resource Pool" column should be sorted in descending order

  Scenario: Sort resource pools by ascending number of CPU cores
    When I am on the resource pool page
      And I sort by "CPU Cores" in ascending order
    Then the "CPU Cores" column should be sorted in ascending order
  
  Scenario: Sort resource pools by descending number of CPU cores
    When I am on the resource pool page
      And I sort by "CPU Cores" in descending order
    Then the "CPU Cores" column should be sorted in descending order
  
  Scenario: Sort resource pools by descending memory usage
    When I am on the resource pool page
      And I sort by "Memory Usage" in descending order
    Then the "Memory Usage" column should be sorted in descending order
  
  Scenario: Sort resource pools by ascending memory usage
    When I am on the resource pool page
      And I sort by "Memory Usage" in ascending order
    Then the "Memory Usage" column should be sorted in ascending order
  
  Scenario: Sort resource pools by ascending creation time
    When I am on the resource pool page
      And I sort by "Created" in ascending order
    Then the "Created" column should be sorted in ascending order
  
  Scenario: Sort resource pools by descending creation time
    When I am on the resource pool page
      And I sort by "Created" in descending order
    Then the "Created" column should be sorted in descending order
  
  Scenario: Sort resource pools by descending modification time
    When I am on the resource pool page
      And I sort by "Last Modified" in descending order
    Then the "Last Modified" column should be sorted in descending order

  @clean_pools
  Scenario: Sort resource pools by ascending modification time
    When I am on the resource pool page
      And I sort by "Last Modified" in ascending order
    Then the "Last Modified" column should be sorted in ascending order
