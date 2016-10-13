require_relative './cli'


include ::CCApi

#
# Returns the same CLI or UI instance across all
# test scenarios.
#
class CC
    @@cli = CLI.new
    def self.CLI
        return @@cli
    end
end