# Coding Guidelines for Acceptance Tests
The Cucumber/Capybara tests are code, and as such they should follow certain best-practices like other code.

## Coding Style
 * Indent with spaces, no tabs
 * Avoid trailing whitespace
 * Indent 4 spaces for .rb files
 * Indent 2 spaces for Feature files

## Best Practices
 * Each feature file should be independent of all other feature files. In other words, the tests for a single feature can be run in isolation from other features.
 * Each feature should have a tag name unique to that feature
 * Wherever possible, scenarios within a feature should have no dependencies on other scenarios in the same feature.
   * For cases where several scenarios require the same setup, those scenarios should be put in their own feature file, and they should use Cucumber’s Background capability to define the common setup logic.
   * Any exception cases where one scenario depends on another should be clearly documented.
 * Wherever possible, each scenario should have an associated cleanup action such that the system is left in the same state as it was at the start of the scenario.
   * Cleanup actions should be implemented as tagged hooks to avoid cluttering the scenarios.
   * These resource-specific cleanup actions should be implemented in the resource-specific step-definition file (e.g. the implementation of @cleanup_hosts belongs be in acceptance/ui/features/steps/hosts.rb).
   * For performance reasons, it is OK to use the CLI to perform cleanup even though the test is UI-based.
 * Do not hardcode input, environment-specific values in any step definitions (acceptance/ui/features/steps/*). All input or environment-specific values should be defined in feature files or externalized in an input data file (see “Externalize Input Data” below).
 * Do not hardcode DOM references in any feature definitions or step definitions. All DOM references should be defined in a page definition (acceptance/ui/features/pages/*).
 * Avoid conjunctive step definitions/Reuse steps wherever possible - one action per step makes the step definitions easier to write, maintain and reuse. Reusing steps makes it easier to add feature/scenarios without writing more Ruby code.

## Externalize Input Data
For simplicity’s sake, the first round of acceptance tests included several literal values that are only valid in the context of a specific developer’s environment. Here are some examples:

```
When I fill in the Host field with "gjones-dev"
When I fill in the Host field with "vagrant"
When I fill in the Host field with "172.17.42.1"
When I remove "roei-dev"
Then I should see "roei-dev" in the "Name" column
Then I should see "Username: zenoss"
```

This practice makes it very difficult, if not impossible, to run the tests in different environments.  One of the first steps in that direction is to externalize the input data so environment-specific details are not hard-coded into the tests themselves.

To do this, we can use a custom URL to refer to arbitrary data loaded from different dictionaries. The URL format is

```
    table:://<tableType>/<tableName>/<propertyName>
```

where

 * `<tableType>`	specifies the type of table table ("hosts", "pools", etc)
 * `<tableName>`	specifies a unique instance of a table (e.g. a host or pool name)
 * `<propertyName>`	specifies the name of a property on the table instance

For example, the following step:

```
    When I fill in the Host field with "vagrant"
```

Can be replaced with:

```
    When I fill in the Host field with "table://hosts/host1/hostName"
```

In each step definition where a value needs to be externalized, the step definition will call a common routine, `getTableValue()`, passing it the value from step statement. The `getTableValue()` routine which will either:

 * return the input value as is, if the input value does not start with “table://”, or
 * look up the value specified by the URL using a table of data loaded at startup, or
 * throw an error if any one of `<tableType>`, `<tableName>`, `<propertyName>` is undefined

Here is an example step definition with a pseudo-code implementation of `getTableValue()`:

```

When(/^I fill in the Host field with "(.*?)"$/) do |valueOrTableUrl|
    hostName = getTableValue(valueOrTableUrl)
    @hosts_page.hostName_input.set hostName
end

def getTableValue(valueOrTableUrl)
    if valueOrTableUrl does not begin with "table://"
        return valueOrTableUrl

    if tableType from valueOrTableUrl not defined
        raise(ArgumentError.new('Invalid table type')))

    if tableName from valueOrTableUrl not defined
        raise(ArgumentError.new('Invalid table name')))

    if propertyName from valueOrTableUrl not defined
        raise(ArgumentError.new('Invalid property name')))

    return tables[tableType][tableName][propertyName]
end
```

The tables of data can be read from JSON files at startup.  Consider the following (abbreviated) example of a `hosts.json` data file:

```

{
  "hosts" {
    "defaultHost": {
      “hostName": "gjones-dev"
      "rpcPort": "4979"
      "name": "gjones-dev",
      "pool": "default",
    },

    "host2" : {
      “hostName": "vagrant1"
      "rpcPort": "4979"
      "name": "vagrant1",
      "pool": "default",
    }
  }
}
```
The value of `table://hosts/defaultHost/“hostName` would be "gjones-dev"

The JSON files should be stored in a directory named `acceptance/ui/features/data/<dataset>`  where `<dataset>` is the name of a set of data. Separating the data files into different directories by dataset will allow the same tests to be executed against different deployments. Each kind of table type (hosts, pools, etc) should be defined a separate JSON file. The file contents should be a set of objects keyed by object name.  An optional `--dataset` parameter should be added to the `runUIAcceptance.sh` script to allow the user to specify different datasets.  The value of the parameter can be passed into the container as an environment variable. If not specified, the tests should default to the dataset named “default”.



## References
 * [Top 5 Cucumber Best Practices](http://blog.codeship.com/cucumber-best-practices/)
 * [15 Expert Tips For Using Cucumber](https://blog.engineyard.com/2009/15-expert-tips-for-using-cucumber)
