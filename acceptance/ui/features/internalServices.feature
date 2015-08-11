@internalServices @screenshot
Feature: Internal Services
  In order to use Control Center
  As a CC admin user
  I want to validate Internal Services

  Background:
    Given that the admin user is logged in

  Scenario: Check that Internal Services is active
    When I am on the applications page
    Then "Internal Services" should be active
      And I should see "Internal" in the "Deployment ID" column

  Scenario: View Internal Services page
    When I am on the applications page
      And I view the details for "Internal Services" in the "Applications" table
    Then I should see "Services"
      And I should see an entry for "Celery" in the table
      And I should see an entry for "Docker Registry" in the table
      And I should see an entry for "Elastic Search - LogStash" in the table
      And I should see an entry for "Elastic Search - Serviced" in the table
      And I should see an entry for "Logstash" in the table
      And I should see an entry for "OpenTSDB" in the table
      And I should see an entry for "Zookeeper" in the table
      And I should see "Graphs"
      And I should see "CPU Usage"
      And I should see "Memory Usage"

  Scenario: View Settings drop-down
    When I am on the applications page
      And I view the details for "Internal Services" in the "Applications" table
    Then I should see "Settings"
    When I click on "Settings"
    Then I should see "Range"
      And I should see "Aggregator"
      And I should see "Refresh"

  Scenario: Check that all services are active
    When I am on the applications page
      And I view the details for "Internal Services" in the "Applications" table
    Then "Celery" should be active
      And "Docker Registry" should be active
      And "Elastic Search - LogStash" should be active
      And "Elastic Search - Serviced" should be active
      And "Logstash" should be active
      And "OpenTSDB" should be active
      And "Zookeeper" should be active

  Scenario: View the CPU Usage graph
    When I am on the applications page
      And I view the details for "Internal Services" in the "Applications" table
    Then I should see "CPU (System)" in the "CPU Usage" graph
      And I should see "CPU (User)" in the "CPU Usage" graph
      And I should see "Total % Used" in the "CPU Usage" graph
    When I hover over the "CPU Usage" graph
    Then I should see "CPU (System)" in the hover box
      And I should see "CPU (User)" in the hover box

  Scenario: View the Memory Usage graph
    When I am on the applications page
      And I view the details for "Internal Services" in the "Applications" table
    Then I should see "Total bytes" in the "Memory Usage" graph
    When I hover over the "Memory Usage" graph
    Then I should see "Memory Usage" in the hover box

  Scenario: View details for the Celery service
    When I am on the applications page
      And I view the details for "Internal Services" in the "Applications" table
      And I view the details for "Celery" in the "Services" table
    Then I should not see an entry for "OpenTSDB" in the table
      And I should see "Total % Used" in the "CPU Usage" graph
      And I should see "Total bytes" in the "Memory Usage" graph

  Scenario: View details for the Docker Registry service
    When I am on the applications page
      And I view the details for "Internal Services" in the "Applications" table
      And I view the details for "Docker Registry" in the "Services" table
    Then I should not see an entry for "Elastic Search - Serviced" in the table
      And I should see "Total % Used" in the "CPU Usage" graph
      And I should see "Total bytes" in the "Memory Usage" graph

  Scenario: View details for the Elastic Search - LogStash service
    When I am on the applications page
      And I view the details for "Internal Services" in the "Applications" table
      And I view the details for "Elastic Search - LogStash" in the "Services" table
    Then I should not see an entry for "Celery" in the table
      And I should see "Total % Used" in the "CPU Usage" graph
      And I should see "Total bytes" in the "Memory Usage" graph

  Scenario: View details for the Elastic Search - Serviced service
    When I am on the applications page
      And I view the details for "Internal Services" in the "Applications" table
      And I view the details for "Elastic Search - Serviced" in the "Services" table
    Then I should not see an entry for "Docker Registry" in the table
      And I should see "Total % Used" in the "CPU Usage" graph
      And I should see "Total bytes" in the "Memory Usage" graph

  Scenario: View details for the Logstash service
    When I am on the applications page
      And I view the details for "Internal Services" in the "Applications" table
      And I view the details for "Logstash" in the "Services" table
    Then I should not see an entry for "Zookeeper" in the table
      And I should see "Total % Used" in the "CPU Usage" graph
      And I should see "Total bytes" in the "Memory Usage" graph

  Scenario: View details for the OpenTSDB service
    When I am on the applications page
      And I view the details for "Internal Services" in the "Applications" table
      And I view the details for "OpenTSDB" in the "Services" table
    Then I should not see an entry for "Elastic Search - LogStash" in the table
      And I should see "Total % Used" in the "CPU Usage" graph
      And I should see "Total bytes" in the "Memory Usage" graph

  Scenario: View details for the Zookeeper service
    When I am on the applications page
      And I view the details for "Internal Services" in the "Applications" table
      And I view the details for "Zookeeper" in the "Services" table
    Then I should not see an entry for "Celery" in the table
      And I should see "Total % Used" in the "CPU Usage" graph
      And I should see "Total bytes" in the "Memory Usage" graph
