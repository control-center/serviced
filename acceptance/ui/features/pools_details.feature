@pools_details @screenshot
Feature: Resource Pool Details
  In order to use Control Center
  As a CC admin user
  I want to view resource pool details

  Background:
    Given that the admin user is logged in
      And that the default resource pool is added

  Scenario: View resource pool details page
    When I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name" in the "Resource Pools" table
    Then I should see "Virtual IPs"
      And I should see "IP"
      And I should see "Netmask"
      And I should see "Bind Interface"
      And I should see "Action"
      And I should see "Hosts"

  @clean_hosts
  Scenario: View default resource pool details
    Given only the default host is added
    When I am on the hosts page
      And I add the "host2" host
      And I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name" in the "Resource Pools" table
    Then the details for "Resource Pool" should be "table://pools/defaultPool/name"
      And the details for "CPU Cores" should be the sum of "table://hosts/defaultHost/cores, table://hosts/host2/cores"
      And the details for "Memory" should be the sum of "table://hosts/defaultHost/memoryGB, table://hosts/host2/memoryGB"

  @clean_hosts
  Scenario: View host details in the resource pool details
    Given only the default host is added
    When I am on the hosts page
      And I add the "host2" host
      And I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name" in the "Resource Pools" table
    Then I should see "table://hosts/defaultHost/name" in the "Name" column
      And I should see "table://hosts/host2/name" in the "Name" column
      And I should see "table://hosts/defaultHost/memoryGB" in the "Memory" column
      And I should see "table://hosts/host2/memoryGB" in the "Memory" column
      And I should see "table://hosts/defaultHost/cores" in the "CPU Cores" column
      And I should see "table://hosts/host2/cores" in the "CPU Cores" column
      And I should see "table://hosts/defaultHost/kernelVersion" in the "Kernel Version" column
      And I should see "table://hosts/host2/kernelVersion" in the "Kernel Version" column

  Scenario: View Add Virtual IP dialog
    When I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name" in the "Resource Pools" table
      And I click the Add Virtual IP button
    Then I should see "Add Virtual IP"
      And I should see the IP field
      And I should see the Netmask field
      And I should see the Interface field

  Scenario: Add a virtual IP with a too-long interface name
    When I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name" in the "Resource Pools" table
      And I click the Add Virtual IP button
      And I fill in the IP field with "table://virtualips/ip1/ip"
      And I fill in the Netmask field with "table://virtualips/ip1/netmask"
      And I fill in the Interface field with "tooLongInterfaceName"
      And I add the virtual IP
    Then I should see "Adding pool virtual ip failed"
      And I should see "Internal Server Error: virtual ip name too long"

  Scenario: Add a virtual IP with an invalid IP address
    When I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name" in the "Resource Pools" table
      And I click the Add Virtual IP button
      And I fill in the IP field with "bogusvirtualip"
      And I fill in the Netmask field with "table://virtualips/ip1/netmask"
      And I fill in the Interface field with "table://virtualips/ip1/interface"
      And I add the virtual IP
    Then I should see "Adding pool virtual ip failed"
      And I should see "Internal Server Error: invalid IP Address"

  Scenario: Add a virtual IP with an invalid netmask
    When I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name" in the "Resource Pools" table
      And I click the Add Virtual IP button
      And I fill in the IP field with "table://virtualips/ip1/ip"
      And I fill in the Netmask field with "bogusnetmask"
      And I fill in the Interface field with "table://virtualips/ip1/interface"
      And I add the virtual IP
    Then I should see "Adding pool virtual ip failed"
      And I should see "Internal Server Error: invalid IP Address"

  @clean_virtualips
  Scenario: Add a valid virtual IP
    Given that the "table://pools/defaultPool/name" pool has no virtual IPs
    When I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name" in the "Resource Pools" table
      And I click the Add Virtual IP button
      And I fill in the IP field with "table://virtualips/ip1/ip"
      And I fill in the Netmask field with "table://virtualips/ip1/netmask"
      And I fill in the Interface field with "table://virtualips/ip1/interface"
      And I add the virtual IP
    Then I should see "Added new pool virtual ip"
      And I should see an entry for "table://virtualips/ip1/ip" in the table

  Scenario: Remove a virtual IP
    Given that the "ip1" virtual IP is added to the "table://pools/defaultPool/name" pool
    When I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name" in the "Resource Pools" table
    Then I should see an entry for "table://virtualips/ip1/ip" in the table
    When I remove "table://virtualips/ip1/ip"
    Then I should see "This action will permanently delete the virtual IP"
    When I click "Remove Virtual IP"
    Then I should not see an entry for "table://virtualips/ip1/ip" in the table

  @clean_virtualips
  Scenario: Add another virtual IP
    Given that the "ip1" virtual IP is added to the "table://pools/defaultPool/name" pool
    When I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name" in the "Resource Pools" table
      And I click the Add Virtual IP button
      And I fill in the IP field with "table://virtualips/ip2/ip"
      And I fill in the Netmask field with "table://virtualips/ip2/netmask"
      And I fill in the Interface field with "table://virtualips/ip2/interface"
      And I add the virtual IP
    Then I should see "Added new pool virtual ip"
      And I should see an entry for "table://virtualips/ip1/ip" in the table
      And I should see an entry for "table://virtualips/ip2/ip" in the table

  @clean_virtualips @clean_pools
  Scenario: Add a virtual IP to another resource pool
    Given that the "table://pools/pool2/name" pool is added
    When I am on the resource pool page
      And I view the details of "table://pools/pool2/name" in the "Resource Pools" table
      And I click the Add Virtual IP button
      And I fill in the IP field with "table://virtualips/ip2/ip"
      And I fill in the Netmask field with "table://virtualips/ip2/netmask"
      And I fill in the Interface field with "table://virtualips/ip2/interface"
      And I add the virtual IP
    Then I should see "Added new pool virtual ip"
      And I should see an entry for "table://virtualips/ip2/ip" in the table

  @clean_virtualips
  Scenario: Add a duplicate virtual IP
    Given that the "ip1" virtual IP is added to the "table://pools/defaultPool/name" pool
    When I am on the resource pool page
      And I view the details of "table://pools/defaultPool/name" in the "Resource Pools" table
      And I click the Add Virtual IP button
      And I fill in the IP field with "table://virtualips/ip1/ip"
      And I fill in the Netmask field with "table://virtualips/ip1/netmask"
      And I fill in the Interface field with "table://virtualips/ip1/interface"
      And I add the virtual IP
    Then I should see "Adding pool virtual ip failed"
      And I should see "Internal Server Error: facade: ip exists in resource pool"
    When I close the dialog
    Then I should see an entry for "table://virtualips/ip1/ip" in the table
