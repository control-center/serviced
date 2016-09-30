@login-required
Feature: V2 Services tests

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
    Given PENDING
    #When I grab the ID of a parent service as "serviceid"
    #And I send a GET request to CC at "/api/v2/services/{serviceid}/services"
    #And the JSON response root should be object
    #And the JSON response should have key "Instances"

  @services
  Scenario: GET IP assignments for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$[0].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/services/{serviceid}/ipassignments"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].IPAddress"

  @services
  Scenario: GET context for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    Given PENDING
    #When I grab "$[0].ID" as "serviceid"
    #And I send a GET request to CC at "/api/v2/services/{serviceid}/ipassignments"
    #Then the response status should be "200"
    #And the JSON response root should be array
    #And the JSON response should have key "$[0].IPAddress"

  @services
  Scenario: GET IP assignments for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$[0].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/services/{serviceid}/publicendpoints"
    Then the response status should be "200"
    And the JSON response root should be array
    # can be an empty array, need to find one with an endpoint to query into it
    #And the JSON response should have key "$[0].IPAddress"

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
    Given PENDING
    # need to be able to find a service where HasChildren = true
    #When I grab "$[0].ID" as "serviceid"
    #And I send a GET request to CC at "/api/v2/services/{serviceid}/instances"
    #Then the response status should be "200"
    #And the JSON response root should be array
    #And the JSON response should have key "$[0].InstanceID"

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
    When I grab "$[0].ID" as "serviceid"
    And I send a GET request to CC at "/api/v2/services/{serviceid}/serviceconfigs"
    Then the response status should be "200"
    When I grab "$[0].ID" as "serviceconfigid"
    And the JSON response root should be array
    Given PENDING
    # Need to find a service config with files to verify
    #And the JSON response should have key "$[0].Owner"

  @services
  Scenario:
  Scenario: POST a service config for a service
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    When I grab "$[0].ID" as "serviceid"
    # And I send a POST request to CC at "/api/v2/services/{serviceid}/serviceconfigs" with body "{"Filename":"/opt/foo","Permissions":"644","Owner":"zenoss:zenoss","Content":"some content"}"
    Then the response status should be "200"
    # ugh this is going...wherever. need to use old API to create a service to stick the config on
