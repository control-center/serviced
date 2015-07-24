@pools_details
Feature: Resource Pool Details
  In order to use Control Center
  As a CC admin user
  I want to view resource pool details

  Background:
    Given that the admin user is logged in
      And that the default resource pool exists

  Scenario: View resource pool details page
    When I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name"
    Then I should see "Virtual IPs"
      And I should see "IP"
      And I should see "Netmask"
      And I should see "Bind Interface"
      And I should see "Action"
      And I should see "Hosts"

  @clean_hosts
  Scenario: View default resource pool details
    Given only the default host is defined
    When I am on the hosts page
      And I add the "host2" host
      And I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name"
    Then the details for "Resource Pool" should be "table://pools/defaultPool/name"
      And the details for "CPU Cores" should be the sum of "table://hosts/defaultHost/cores, table://hosts/host2/cores"
      And the details for "Memory" should be the sum of "table://hosts/defaultHost/memoryGB, table://hosts/host2/memoryGB"

  @clean_hosts
  Scenario: View host details in the resource pool details
    Given only the default host is defined
    When I am on the hosts page
      And I add the "host2" host
      And I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name"
    Then I should see "table://hosts/defaultHost/name" in the "Name" column
      And I should see "table://hosts/host2/name" in the "Name" column
      And I should see "table://hosts/defaultHost/memoryGB" in the "Memory" column
      And I should see "table://hosts/host2/memoryGB" in the "Memory" column
      And I should see "table://hosts/defaultHost/cores" in the "CPU Cores" column
      And I should see "table://hosts/host2/cores" in the "CPU Cores" column
      And I should see "table://hosts/defaultHost/kernelVersion" in the "Kernel Version" column
      And I should see "table://hosts/host2/kernelVersion" in the "Kernel Version" column
