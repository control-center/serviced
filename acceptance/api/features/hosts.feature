@login-required
Feature: blam

Scenario: Basic authentication
	  Given I send and accept JSON
	  When I send a GET request to CC at "https://10.87.128.95/hosts"
	  Then the response status should be "200"
	  





