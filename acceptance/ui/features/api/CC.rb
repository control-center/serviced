require_relative './cli'
require_relative './ui'

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

    @@ui = UI.new
    def self.UI
        return @@ui
    end
end