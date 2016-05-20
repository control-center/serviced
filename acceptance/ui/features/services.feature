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
  Scenario: Public Endpoint dialog - check all labels
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
      And I click "Cancel"

  @services_endpoint
  Scenario: Public Endpoint dialog - VHost mode visible fields
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I choose VHost type
    Then I should see the Add Public Endpoint dialog
      And I should see VHost fields
      And I click "Cancel"

  @services_endpoint
  Scenario: Public Endpoint dialg - Port mode visible fields
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
    Then I should see Port fields
      And I click "Cancel"

  @services_endpoint
  Scenario: Public Endpoint dialog - no port given
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I click "Add and Restart Service"
    Then I should see "Missing port"
      And I click "Cancel"

  @services_endpoint
  Scenario: Public Endpoint dialog - bad port given
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I fill in Port "99999"
      And I click "Add and Restart Service"
    Then I should see "Port must be between 1 and 65536"
      And I click "Cancel"

  @services_endpoint
  Scenario: Public Endpoint dialog - bad host given
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I fill in Host "a.b.c.d"
      And I fill in Port "23456"
      And I click "Add and Restart Service"
    Then I should see "no such host"
      And I click "Cancel"

  @services_endpoint
  Scenario: Public Endpoint dialog - add https
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I select Service "table://services/childServiceWithVHost/name" - "table://services/childServiceWithVHost/endpoint_https"
      And I fill in Host "table://services/childServiceWithVHost/host"
      And I fill in Port "table://services/childServiceWithVHost/port_https"
      And I select Protocol "table://services/childServiceWithVHost/protocol_https"
      And I click "Add and Restart Service"
    Then I should see "table://services/childServiceWithVHost/name"
      And I should see "Public Endpoints"
      And Endpoint details should be service:"table://services/childServiceWithVHost/name" endpoint:"table://services/childServiceWithVHost/endpoint_https" type:"port" protocol:"table://services/childServiceWithVHost/protocol_https_o" host:"table://services/childServiceWithVHost/host" port:"table://services/childServiceWithVHost/port_https"

  @services_endpoint
  Scenario: Public Endpoint dialog - add http
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I select Service "table://services/childServiceWithVHost/name" - "table://services/childServiceWithVHost/endpoint_http"
      And I fill in Host "table://services/childServiceWithVHost/host"
      And I fill in Port "table://services/childServiceWithVHost/port_http"
      And I select Protocol "table://services/childServiceWithVHost/protocol_http"
      And I click "Add and Restart Service"
    Then I should see "table://services/childServiceWithVHost/name"
      And I should see "Public Endpoints"
      And Endpoint details should be service:"table://services/childServiceWithVHost/name" endpoint:"table://services/childServiceWithVHost/endpoint_http" type:"port" protocol:"table://services/childServiceWithVHost/protocol_http_o" host:"table://services/childServiceWithVHost/host" port:"table://services/childServiceWithVHost/port_http"

  @services_endpoint
  Scenario: Public Endpoint dialog - add tls
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I select Service "table://services/childServiceWithVHost/name" - "table://services/childServiceWithVHost/endpoint_tls"
      And I fill in Host "table://services/childServiceWithVHost/host"
      And I fill in Port "table://services/childServiceWithVHost/port_tls"
      And I select Protocol "table://services/childServiceWithVHost/protocol_tls"
      And I click "Add and Restart Service"
    Then I should see "table://services/childServiceWithVHost/name"
      And I should see "Public Endpoints"
      And Endpoint details should be service:"table://services/childServiceWithVHost/name" endpoint:"table://services/childServiceWithVHost/endpoint_tls" type:"port" protocol:"table://services/childServiceWithVHost/protocol_tls_o" host:"table://services/childServiceWithVHost/host" port:"table://services/childServiceWithVHost/port_tls"

  @services_endpoint
  Scenario: Public Endpoint dialog - add other
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I select Service "table://services/childServiceWithVHost/name" - "table://services/childServiceWithVHost/endpoint_other"
      And I fill in Host "table://services/childServiceWithVHost/host"
      And I fill in Port "table://services/childServiceWithVHost/port_other"
      And I select Protocol "table://services/childServiceWithVHost/protocol_other"
      And I click "Add and Restart Service"
    Then I should see "table://services/childServiceWithVHost/name"
      And I should see "Public Endpoints"
      And Endpoint details should be service:"table://services/childServiceWithVHost/name" endpoint:"table://services/childServiceWithVHost/endpoint_other" type:"port" protocol:"table://services/childServiceWithVHost/protocol_other_o" host:"table://services/childServiceWithVHost/host" port:"table://services/childServiceWithVHost/port_other"

  @services_endpoint
  Scenario: Public Endpoint dialog - remove https
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I delete Endpoint "table://services/childServiceWithVHost/port_https"
    Then I should not see "table://services/childServiceWithVHost/port_https"

  @services_endpoint
  Scenario: Public Endpoint dialog - remove http
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I delete Endpoint "table://services/childServiceWithVHost/port_http"
    Then I should not see "table://services/childServiceWithVHost/port_http"

  @services_endpoint
  Scenario: Public Endpoint dialog - remove tls
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I delete Endpoint "table://services/childServiceWithVHost/port_tls"
    Then I should not see "table://services/childServiceWithVHost/port_tls"

  @services_endpoint
  Scenario: Public Endpoint dialog - remove other
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I delete Endpoint "table://services/childServiceWithVHost/port_other"
    Then I should not see "table://services/childServiceWithVHost/port_other"

  @services_endpoint_na
  Scenario: Public Endpoint dialog - Re-add https again
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I select Service "table://services/childServiceWithVHost/name" - "table://services/childServiceWithVHost/endpoint_https"
      And I fill in Host "table://services/childServiceWithVHost/host"
      And I fill in Port "table://services/childServiceWithVHost/port_https"
      And I select Protocol "table://services/childServiceWithVHost/protocol_https"
      And I click "Add and Restart Service"
    Then I should see "table://services/childServiceWithVHost/name"
      And I should see "Public Endpoints"
      And Endpoint details should be service:"table://services/childServiceWithVHost/name" endpoint:"table://services/childServiceWithVHost/endpoint_https" type:"port" protocol:"table://services/childServiceWithVHost/protocol_https_o" host:"table://services/childServiceWithVHost/host" port:"table://services/childServiceWithVHost/port_https"

  @services_endpoint_na
  Scenario: Public Endpoint dialog - remove https
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I delete Endpoint "table://services/childServiceWithVHost/port_https"
    Then I should not see "table://services/childServiceWithVHost/port_https"

