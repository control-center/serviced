@services_endpoint @screenshot
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

  Scenario: Public Endpoint dialog - VHost mode visible fields
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I choose VHost type
    Then I should see the Add Public Endpoint dialog
      And I should see VHost fields

  @testme
  Scenario: Public Endpoint dialg - Port mode visible fields
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
    Then I should see Port fields
    And I should see Port "placeholder" to be "table://services/childServiceWithVHost/port_placeholder"
    And I should see Host "placeholder" to be "table://services/childServiceWithVHost/host_placeholder"
    And "table://services/childServiceWithVHost/svc_default" should be selected for Service Endpoint
    And "table://services/childServiceWithVHost/protocol_default" should be selected for Protocol

  @testme
  Scenario: Public Endpoint dialog - no port given
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I click "Add and Restart Service"
    Then I should see "Missing port"

  @testme
  Scenario: Public Endpoint dialog - bad port given
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I fill in Port "99999"
      And I click "Add and Restart Service"
    Then I should see "Port must be between 1 and 65536"


  Scenario: Public Endpoint dialog - bad host given
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I click "Add Public Endpoint"
      And I should see the Add Public Endpoint dialog
      And I fill in Host "a.b.c.d"
      And I fill in Port "table://services/childServiceWithVHost/port_placeholder"
      And I click "Add and Restart Service"
    Then I should see "no such host"

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
      And I should see only one endpoint entry of Port "table://services/childServiceWithVHost/port_https"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_https" and "table://services/childServiceWithVHost/name"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_https" and "table://services/childServiceWithVHost/endpoint_https"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_https" and "port"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_https" and "table://services/childServiceWithVHost/protocol_https_display"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_https" and "table://services/childServiceWithVHost/host"

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
      And I should see only one endpoint entry of Port "table://services/childServiceWithVHost/port_http"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_http" and "table://services/childServiceWithVHost/name"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_http" and "table://services/childServiceWithVHost/endpoint_http"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_http" and "port"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_http" and "table://services/childServiceWithVHost/protocol_http_display"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_http" and "table://services/childServiceWithVHost/host"

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
      And I should see only one endpoint entry of Port "table://services/childServiceWithVHost/port_tls"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_tls" and "table://services/childServiceWithVHost/name"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_tls" and "table://services/childServiceWithVHost/endpoint_tls"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_tls" and "port"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_tls" and "table://services/childServiceWithVHost/protocol_tls_display"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_tls" and "table://services/childServiceWithVHost/host"

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
      And I should see only one endpoint entry of Port "table://services/childServiceWithVHost/port_other"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_other" and "table://services/childServiceWithVHost/name"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_other" and "table://services/childServiceWithVHost/endpoint_other"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_other" and "port"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_other" and "table://services/childServiceWithVHost/protocol_other_display"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_other" and "table://services/childServiceWithVHost/host"

  Scenario: Public Endpoint dialog - remove https
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I delete Endpoint "table://services/childServiceWithVHost/port_https"
    Then I should not see "table://services/childServiceWithVHost/port_https"

  Scenario: Public Endpoint dialog - remove http
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I delete Endpoint "table://services/childServiceWithVHost/port_http"
    Then I should not see "table://services/childServiceWithVHost/port_http"

  Scenario: Public Endpoint dialog - remove tls
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I delete Endpoint "table://services/childServiceWithVHost/port_tls"
    Then I should not see "table://services/childServiceWithVHost/port_tls"

  Scenario: Public Endpoint dialog - remove other
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I delete Endpoint "table://services/childServiceWithVHost/port_other"
    Then I should not see "table://services/childServiceWithVHost/port_other"

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
      And I should see only one endpoint entry of Port "table://services/childServiceWithVHost/port_https"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_https" and "table://services/childServiceWithVHost/name"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_https" and "table://services/childServiceWithVHost/endpoint_https"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_https" and "port"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_https" and "table://services/childServiceWithVHost/protocol_https_display"
      And I should see the endpoint entry with both "table://services/childServiceWithVHost/port_https" and "table://services/childServiceWithVHost/host"

  Scenario: Public Endpoint dialog - remove https
    When I view the details of "table://services/topService/name" in the "Applications" table
      And I view the details of "table://services/childServiceWithVHost/name" in the "Services" table
      And I delete Endpoint "table://services/childServiceWithVHost/port_https"
    Then I should not see "table://services/childServiceWithVHost/port_https"

