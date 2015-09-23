@hosts_sorting @screenshot
Feature: Host Sorting
  In order to use Control Center
  As a CC admin user
  I want to sort hosts according to various attributes

  Background:
    Given that the admin user is logged in
      And that multiple resource pools have been added
      And that multiple hosts have been added

  Scenario: Test ascending name sort
    When I am on the hosts page
      And I sort by "Name" in ascending order
    Then the "Name" column should be sorted in ascending order

  Scenario: Test descending name sort
    When I am on the hosts page
      And I sort by "Name" in descending order
    Then the "Name" column should be sorted in descending order

  Scenario: Test ascending status sort
    When I am on the hosts page
      And I sort by "Active" in ascending order
    Then the "Active" column should be sorted with active hosts on the bottom

  Scenario: Test descending status sort
    When I am on the hosts page
      And I sort by "Active" in descending order
    Then the "Active" column should be sorted with active hosts on top

  Scenario: Test descending resource pool sort
    When I am on the hosts page
      And I sort by "Resource Pool" in descending order
    Then the "Resource Pool" column should be sorted in descending order

  Scenario: Test ascending resource pool sort
    When I am on the hosts page
      And I sort by "Resource Pool" in ascending order
    Then the "Resource Pool" column should be sorted in ascending order

  Scenario: Test descending memory sort
    When I am on the hosts page
      And I sort by "Memory" in descending order
    Then the "Memory" column should be sorted in descending order

  Scenario: Test ascending memory sort
    When I am on the hosts page
      And I sort by "Memory" in ascending order
    Then the "Memory" column should be sorted in ascending order

  Scenario: Test ascending CPU cores sort
    When I am on the hosts page
      And I sort by "CPU Cores" in ascending order
    Then the "CPU Cores" column should be sorted in ascending order

  Scenario: Test descending CPU cores sort
    When I am on the hosts page
      And I sort by "CPU Cores" in descending order
    Then the "CPU Cores" column should be sorted in descending order

  Scenario: Test ascending kernel version sort
    When I am on the hosts page
      And I sort by "Kernel Version" in ascending order
    Then the "Kernel Version" column should be sorted in ascending order

  Scenario: Test descending kernel version sort
    When I am on the hosts page
      And I sort by "Kernel Version" in descending order
    Then the "Kernel Version" column should be sorted in descending order

  Scenario: Test ascending CC release sort
    When I am on the hosts page
      And I sort by "CC Release" in ascending order
    Then the "CC Release" column should be sorted in ascending order

  @clean_hosts @clean_pools
  Scenario: Test descending CC release sort
    When I am on the hosts page
      And I sort by "CC Release" in descending order
    Then the "CC Release" column should be sorted in descending order
    
