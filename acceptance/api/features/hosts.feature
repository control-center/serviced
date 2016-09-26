@login-required
Feature: V2 Hosts tests

Scenario: Basic GET
	  Given I send and accept JSON
	  When I send a GET request to CC at "/api/v2/hosts"
	  Then the response status should be "200"
	  





