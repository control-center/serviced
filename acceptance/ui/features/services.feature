@services @screenshot
Feature: Service Management
  In order to use Control Center
  As a CC admin user
  I want to manage deployed services

  Background:
    Given that the admin user is logged in
      And that the default resource pool is added
      And that the test template is added
      And only the default host is added
      And that the "table://applications/defaultApp/template" application with the "table://applications/defaultApp/id" Deployment ID is added
      And I am on the applications page

  Scenario: View the tenant service page
    When I view the details of "table://services/topService/name" in the "Applications" table
    Then I should see "table://services/topService/name"
      And I should see "IP Assignments"
      And I should see "Configuration Files"
      And I should see "Services"
      And I should see "Scheduled Tasks"
      And I should see "table://services/childServiceNoVHosts/name"
      And I should see "table://services/childServiceWithVHost/name"

  Scenario: View a service page that has public endpoints
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
    Then I should see "table://services/childServiceWithVHost/name"
      And I should see "Public Endpoints"
      And VHost "table://services/childServiceWithVHost/vhostName" should exist
      And Public Port "table://services/childServiceWithVHost/publicPortNumber" should exist
      And I should see "table://services/childServiceWithVHost/configFile"

  @services_endpoint
  Scenario: View the Add Public Endpoint dialog of a service page that has public endpoints
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
    Then I should see the Add Public Endpoint dialog
      And I should see "Type:"
      And I should see "Port"
      And I should see "VHost"
      And I should see "Service - Endpoint:"
      And I should see "Host:"
      And I should see "Port:"
      And I should see "Protocol:"
      And I should find all fields
      And I click "Cancel"

  @services_endpoint
  Scenario: Check VHost mode of the Add Public Endpoint dialog 
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I choose VHost type
    Then I should see the Add Public Endpoint dialog
      And I should see VHost fields
      And I click "Cancel"

  @services_endpoint
  Scenario: Check Port mode of the Add Public Endpoint dialog 
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
    Then I should see Port fields
      And I click "Cancel"
