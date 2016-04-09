@hosts @screenshot
Feature: Host Management
  In order to use Control Center
  As a CC admin user
  I want to manage hosts

  Background:
    Given that the admin user is logged in
      And that the default resource pool is added

  Scenario: View empty Hosts page
    Given there are no hosts added
    When I am on the hosts page
    Then I should see "Hosts Map"
      And I should see "Name"
      And I should see "Active"
      And I should see "Resource Pool"
      And I should see "Memory"
      And I should see "RAM Limit"
      And I should see an empty Hosts page

  Scenario: View Add Host dialog
    When I am on the hosts page
      And I click the add Host button
    Then I should see the Add Host dialog
      And I should see "Host"
      And I should see the Host field
      And I should see "Port"
      And I should see the Port field
      And I should see "Resource Pool ID"
      And I should see the Resource Pool ID field
      And I should see "RAM Limit"
      And I should see the RAM Limit field

  Scenario: Add an invalid host with an invalid port
    Given there are no hosts added
    When I am on the hosts page
      And I click the add Host button
      And I fill in the Host field with "host"
      And I fill in the Port field with "bogusport"
      And I fill in the Resource Pool field with "table://hosts/defaultHost/pool"
      And I fill in the RAM Limit field with "table://hosts/defaultHost/commitment"
      And I click "Add Host"
    Then I should see "Error"
      And I should see "Invalid port number"
      And the Port field should be flagged as invalid
      And I should see an empty Hosts page

  Scenario: Add an invalid host with an empty host
    Given there are no hosts added
    When I am on the hosts page
      And I click the add Host button
      And I fill in the Port field with "4979"
      And I fill in the Resource Pool field with "table://hosts/defaultHost/pool"
      And I fill in the RAM Limit field with "table://hosts/defaultHost/commitment"
      And I click "Add Host"
    Then I should see "Error"
      And I should see "Please enter a valid host name"
      And I should see an empty Hosts page

  Scenario: Add an invalid host with an invalid host name
    Given there are no hosts added
    When I am on the hosts page
      And I click the add Host button
      And I fill in the Host field with "172.17.42.1"
      And I fill in the Port field with "9999"
      And I fill in the Resource Pool field with "table://hosts/defaultHost/pool"
      And I fill in the RAM Limit field with "table://hosts/defaultHost/commitment"
      And I click "Add Host"
    Then I should see "Error"
      And I should see "Bad Request: dial tcp4 172.17.42.1:9999"
      And I should see an empty Hosts page

  Scenario: Add an invalid host with a port out of range
    Given there are no hosts added
    When I am on the hosts page
      And I click the add Host button
      And I fill in the Host field with "172.17.42.1"
      And I fill in the Port field with "75000"
      And I fill in the Resource Pool field with "table://hosts/defaultHost/pool"
      And I fill in the RAM Limit field with "table://hosts/defaultHost/commitment"
      And I click "Add Host"
    Then I should see "Error"
      And I should see "The port number must be between 1 and 65535"
      And I should see an empty Hosts page

  Scenario: Add an invalid host with an invalid RAM Limit field
    Given there are no hosts added
    When I am on the hosts page
      And I click the add Host button
      And I fill in the Host field with "table://hosts/defaultHost/hostName"
      And I fill in the Port field with "table://hosts/defaultHost/rpcPort"
      And I fill in the Resource Pool field with "table://hosts/defaultHost/pool"
      And I fill in the RAM Limit field with "invalidentry"
      And I click "Add Host"
    Then I should see "Error"
      And I should see "Invalid RAM Limit value"
      And I should see an empty Hosts page

  Scenario: Fill in the hosts dialog and cancel
    Given there are no hosts added
    When I am on the hosts page
      And I click the add Host button
      And I fill in the Host field with "table://hosts/defaultHost/hostName"
      And I fill in the Port field with "table://hosts/defaultHost/rpcPort"
      And I fill in the Resource Pool field with "table://hosts/defaultHost/pool"
      And I fill in the RAM Limit field with "table://hosts/defaultHost/commitment"
      And I click "Cancel"
    Then I should see an empty Hosts page
      And I should not see "Success"

  @clean_hosts
  Scenario: Add a valid host
    Given there are no hosts added
    When I am on the hosts page
      And I click the add Host button
      And I fill in the Host field with "table://hosts/defaultHost/hostName"
      And I fill in the Port field with "table://hosts/defaultHost/rpcPort"
      And I fill in the Resource Pool field with "table://hosts/defaultHost/pool"
      And I fill in the RAM Limit field with "table://hosts/defaultHost/commitment"
      And I click "Add Host"
    Then I should see "Success"
      And I should see "table://hosts/defaultHost/name" in the "Name" column
      And I should see "table://hosts/defaultHost/pool" in the "Resource Pool" column
      And I should see "table://hosts/defaultHost/memoryGB" in the "Memory" column
      And I should see "table://hosts/defaultHost/ramGB" in the "RAM Limit" column
      And I should see "table://hosts/defaultHost/cores" in the "CPU Cores" column
      And I should see "Showing 1 Result"

  @clean_hosts
  Scenario: Add another valid host
    Given only the default host is added
    When I am on the hosts page
      And I click the add Host button
      And I fill in the Host field with "table://hosts/host2/hostName"
      And I fill in the Port field with "table://hosts/host2/rpcPort"
      And I fill in the Resource Pool field with "table://hosts/host2/pool"
      And I fill in the RAM Limit field with "table://hosts/host2/commitment"
      And I click "Add Host"
    Then I should see "Success"
      And I should see an entry for "table://hosts/host2/name" in the table
      And I should see "table://hosts/defaultHost/name" in the "Name" column
      And I should see "table://hosts/defaultHost/pool" in the "Resource Pool" column
      And I should see "table://hosts/host2/name" in the "Name" column
      And I should see "table://hosts/host2/pool" in the "Resource Pool" column
      And I should see "table://hosts/host2/memoryGB" in the "Memory" column
      And I should see "table://hosts/host2/ramGB" in the "RAM Limit" column
      And I should see "table://hosts/host2/cores" in the "CPU Cores" column
      And I should see "table://hosts/host2/kernelVersion" in the "Kernel Version" column
      And I should see "table://hosts/host2/ccRelease" in the "CC Release" column
      And I should see "Showing 2 Results"

  @clean_hosts @clean_pools
  Scenario: Add a valid host in a non-default Resource Pool
    Given that the "table://hosts/host3/pool" pool is added
    When I am on the hosts page
      And I click the add Host button
      And I fill in the Host field with "table://hosts/host3/hostName"
      And I fill in the Port field with "table://hosts/host3/rpcPort"
      And I fill in the Resource Pool field with "table://hosts/host3/pool"
      And I fill in the RAM Limit field with "table://hosts/host3/commitment"
      And I click "Add Host"
    Then I should see "Success"
      And I should see an entry for "table://hosts/host3/name" in the table
      And I should see "table://hosts/host3/name" in the "Name" column
      And I should see "table://hosts/host3/pool" in the "Resource Pool" column
      And I should see "table://hosts/host3/memoryGB" in the "Memory" column
      And I should see "table://hosts/host3/ramGB" in the "RAM Limit" column
      And I should see "table://hosts/host3/cores" in the "CPU Cores" column
      And I should see "table://hosts/host3/kernelVersion" in the "Kernel Version" column
      And I should see "table://hosts/host3/ccRelease" in the "CC Release" column

  @clean_hosts
  Scenario: Add a duplicate host
    Given only the default host is added
    When I am on the hosts page
      And I click the add Host button
      And I fill in the Host field with "table://hosts/defaultHost/hostName"
      And I fill in the Port field with "table://hosts/defaultHost/rpcPort"
      And I fill in the Resource Pool field with "table://hosts/defaultHost/pool"
      And I fill in the RAM Limit field with "table://hosts/defaultHost/commitment"
      And I click "Add Host"
    Then I should see "Error"
      And I should see "Internal Server Error: host already exists"
    When I close the dialog
    Then I should see "Showing 1 Result"

  Scenario: Remove a host
    Given only the default host is added
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

  @clean_hosts
  Scenario: View Host Details
    Given only the default host is added
    When I am on the hosts page
      And I view the details of "table://hosts/defaultHost/name" in the "Hosts" table
    Then I should see "Graphs"
      And I should see "CPU Usage"
      And I should see "Load Average"
      And I should see "Memory Usage"
      And I should see "Open File Descriptors"
      And I should see "Memory Major Page Faults"
      And I should see "Paging"
      And I should see "IPs"
      And I should see "Services Instances"

  @clean_hosts
  Scenario: View default host details
    Given only the default host is added
    When I am on the hosts page
      And I view the details of "table://hosts/defaultHost/name" in the "Hosts" table
    Then the details for "Name" should be "table://hosts/defaultHost/hostID"
      And the details for "Resource Pool" should be "table://hosts/defaultHost/pool"
      And the details for "Memory" should be "table://hosts/defaultHost/memoryGB"
      And the details for "CPU Cores" should be "table://hosts/defaultHost/cores"
      And the details for "Kernel Version" should be "table://hosts/defaultHost/kernelVersion"
      And the details for "Kernel Release" should be "table://hosts/defaultHost/kernelRelease"
      And the details for "CC Release" should be "table://hosts/defaultHost/ccRelease"
      And the details for "IP Address" should be "table://hosts/defaultHost/outboundIP"
      And the details for "RAM Limit" should be "table://hosts/defaultHost/ramGB"

  @clean_hosts
  Scenario: View Host Map
    Given only the default host is added
    When I am on the hosts page
      And I add the "host2" host
      And I click the Hosts Map button
    Then I should see "By RAM"
      And I should see "By CPU"
      And I should see "table://hosts/defaultHost/name"
      And I should see "table://hosts/host2/name"
    When I click "By CPU"
    Then I should see "table://hosts/defaultHost/name"
      And I should see "table://hosts/host2/name"
