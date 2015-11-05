@login @screenshot
Feature: User login
  As a Control Center user
  I want to see if the login works

  Scenario: Successful login
    When I am on the login page
      And I fill in the user id field with the default user id
      And I fill in the password field with the default password
      And I click the sign-in button
      And I close the deploy wizard if present
    Then I should see "Applications"
      And I should see "Resource Pools"
      And I should see "Hosts"
      And I should see "Backup / Restore"
      And I should see "Logout"
      And I should see "About"
      And I should see "Application Templates"
      And I should see "Internal Services"

  Scenario: Unsuccessful login
    When I am on the login page
      And I fill in the user id field with "bogus"
      And I fill in the password field with "notarealpassword"
      And I click the sign-in button
    Then I should see the login error "Username/Password is invalid"
