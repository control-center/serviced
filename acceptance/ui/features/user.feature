@user
Feature: User Details
  As a Control Center user
  I want to manage user details

  @login-required
  Scenario: View user details
    When I view user details
    Then I should see "User Details"
      And I should see "Username: zenoss"
      And I should see "Messages"
      And I should see "Clear"

  @login-required @defaultHostPage
  Scenario: View all messages
    When I view user details
    Then I should see my messages

  @login-required @defaultHostPage
  Scenario: Mark messages as read
    When I view user details
      And I click on an unread message
    Then I should see a checkmark

  @login-required @defaultHostPage
  Scenario: Clear messages
    When I view user details
      And I clear my messages
    Then I should not see any messages

  @login-required
  Scenario: Switch language to Spanish
    When I view user details
      And I switch the language to Spanish
    Then I should see "Usuario"
      And I should see "Mensajes"
      And I should see "Servidores"

  @login-required
  Scenario: Switch language to English
    When I view user details
      And I switch the language to English
    Then I should see "Username"
      And I should see "Messages"
      And I should see "Hosts"
