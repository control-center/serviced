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

  @login-required
  Scenario: View all messages
    Given I have messages
    When I view user details
    Then I should see my messages

  @login-required
  Scenario: Mark messages as read
    Given I have messages
    When I view user details
      And I click on the unread message "resource pool exists"
    Then I should see that the "resource pool exists" message is marked as read

  @login-required
  Scenario: Clear messages
    Given I have messages
    When I view user details
      And I clear my messages
    Then I should not see any messages

  @login-required
  Scenario: Switch language to Spanish
    When I view user details
      And I switch the language to Spanish
    Then I should see "Usuario"
      And I should see "Mensajes"
      And I should see "Borrar"
    When I close the dialog
    Then I should see "Apps Instaladas"
      And I should see "Grupo de Recursos"
      And I should see "Servidores"
      And I should see "Registros"

  @login-required
  Scenario: Switch language to English
    When I view user details
      And I switch the language to English
    Then I should see "Username"
      And I should see "Messages"
      And I should see "Clear"
    When I close the dialog
    Then I should see "Applications"
      And I should see "Resource Pools"
      And I should see "Hosts"
      And I should see "Logs"