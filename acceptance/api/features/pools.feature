@login-required
Feature: V2 Pools tests

  Scenario: GET all pools
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/pools"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].MemoryCapacity"


  Scenario: POST should fail
    Given I send and accept JSON
    When I send a POST request to CC at "/api/v2/pools"
    Then the response status should be "405"