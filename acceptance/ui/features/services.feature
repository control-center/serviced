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