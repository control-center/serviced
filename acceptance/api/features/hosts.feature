@login-required @hosts
Feature: V2 Hosts tests

  Background:
    Given that the test template is added
    And that the default resource pool is added
    And that the "testsvc" application with the "Acceptance" Deployment ID is added
    And that the "testsvc" application is started

  Scenario: GET all hosts
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/hosts"
    Then the response status should be "200"
    And the JSON response root should be array

  Scenario: POST should fail
    Given I send and accept JSON
    When I send a POST request to CC at "/api/v2/hosts"
    Then the response status should be "405"

  Scenario: GET instances for host
    Given I send and accept JSON
    When I send a GET request to CC at "/api/v2/hosts"
    Then the response status should be "200"
    When I grab "$[0].ID" as "hostid"
    And I send a GET request to CC at "/api/v2/hosts/{hostid}/instances"
    Then the response status should be "200"
    And the JSON response root should be array
    And the JSON response should have key "$[0].ServiceName"







