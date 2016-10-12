@login-required
Feature: V2 Services tests

  Background:
    Given that the test template is added
    And that the default resource pool is added
    And that the "testsvc" application with the "Acceptance" Deployment ID is added
    And that the "testsvc" application is started

  @services
  Scenario: GET all services
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].PoolID"

  @services
  Scenario: POST should fail
    Given I send and accept JSON
    When I send a POST request to CC at "/api/v2/services"
    Then the response status should be "405"

  @services
  Scenario: GET tenant services
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services?tenants"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].PoolID"

  @services
  Scenario: GET status for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$[0].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/statuses?serviceId={serviceid}"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].DesiredState"

  @services
  Scenario: GET details for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$[0].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/services/{serviceid}"
    Then the response status should be "200"
    And the JSON response root should be object
    And the JSON response should have key "Instances"

  @services
  Scenario: GET children for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$.[?(@["Name"]=="testsvc")].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/services/{serviceid}/services"
    And the JSON response root should be array
    And the JSON response should have key "$[0].Instances"

  @services
  Scenario: GET IP assignments for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$.[?(@["Name"]=="s1")].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/services/{serviceid}/ipassignments"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].IPAddress"
    And the JSON response should have value "1000" at "$[0].Port"

  @services
  Scenario: PUT and GET context for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$.[?(@["Name"]=="s1")].ID" as "serviceid"
    And I send a PUT request from file "default/servicecontext.json" to CC at "/api/v2/services/{serviceid}/context"
    Then the response status should be "200"
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$.[?(@["Name"]=="s1")].ID" as "serviceid"
    When I send a GET request to CC at "/api/v2/services/{serviceid}/context"
    Then the response status should be "200"
    And the JSON response root should be object

  @services
  Scenario: GET Public endpoints for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$.[?(@["Name"]=="s2")].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/services/{serviceid}/publicendpoints"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].Protocol"

  @services
  Scenario: GET monitoring profile for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$[0].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/services/{serviceid}/monitoringprofile"
    Then the response status should be "200"
    And the JSON response root should be object
    And the JSON response should have key "MetricConfigs"

  @services
  Scenario: GET instances for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$.[?(@["Name"]=="s2")].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/services/{serviceid}/instances"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].InstanceID"

  @services
  Scenario: GET service configs for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$[0].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/services/{serviceid}/serviceconfigs"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].ID"


  @services
  Scenario: GET details for a service config
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$.[?(@["Name"]=="s1")].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/services/{serviceid}/serviceconfigs"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have value "/etc/my.cnf" at "$..Filename*"

  @services @reload_service
  Scenario: POST a service config for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$.[?(@["Name"]=="s1")].ID" as "serviceid"
    And I send a POST request from file "default/serviceconfig.json" to CC at "/api/v2/services/{serviceid}/serviceconfigs"
    Then the response status should be "200"

  @services
  Scenario: DELETE a service config from a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$.[?(@["Name"]=="s1")].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/services/{serviceid}/serviceconfigs"
    Then the response status should be "200"
    And the JSON response root should be array
    When I grab "$[0].ID" as "configid"
    And I send a DELETE request to CC at "/api/v2/serviceconfigs/{configid}"
    Then the response status should be "200"
    