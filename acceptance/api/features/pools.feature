@login-required
Feature: V2 Pools tests

  Background:
    Given that the test template is added
    And that the default resource pool is added

  @pools
  Scenario: GET all pools
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/pools"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].MemoryCapacity"

  @pools
  Scenario: POST to GET pools should fail
    Given I send and accept JSON
    When I send a POST request to CC at "/api/v2/pools"
    Then the response status should be "405"

  @pools
  Scenario: GET the list of hosts for a given pool
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/pools"
    Then the response status should be "200"
    When I grab "$[0].ID" as "poolid"
    And I send a GET request to CC at "/api/v2/pools/{poolid}/hosts"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].Memory"

