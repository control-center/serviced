@login-required
Feature: V2 Services tests

  Scenario: GET all services
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].PoolID"

  Scenario: GET tenant services
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/services?tenants"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].PoolID"

  Scenario: POST should fail
    Given I send and accept JSON
    When I send a POST request to CC at "/api/v2/services"
    Then the response status should be "405"