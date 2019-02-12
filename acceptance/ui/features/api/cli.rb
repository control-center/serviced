require "json"
require "capybara/rspec"
require_relative 'cli_host'
require_relative 'cli_pool'
require_relative 'cli_service'
require_relative 'cli_template'

module CCApi
    #
    # Encapsulates the CLI functionality to be exposed to the
    # acceptance tests: CC.CLI.<function>
    #
    class CLI
        include ::RSpec::Matchers

        def initialize()
            @last_command = ""
            @last_output = ""
            @subcommands = {
                host: CLI_Host.new,
                service: CLI_Service.new,
                template: CLI_Template.new,
                pool: CLI_Pool.new,
            }
        end

        # CC.CLI.host.<function>
        def host
            return @subcommands[:host]
        end

        # CC.CLI.service.<function>
        def service
            return @subcommands[:service]
        end

        # CC.CLI.template.<function>
        def template
            return @subcommands[:template]
        end

        # CC.CLI.pool.<function>
        def pool
            return @subcommands[:pool]
        end

        # executes a CLI command, checks the result, and returns the output.
        def execute(command, stderr = true, verify = true)
            servicedCLI = get_serviced_cli()
            command.gsub! "%{serviced}", servicedCLI
            command += " 2>&1" if stderr
            command += " 2>/dev/null" if !stderr
            @last_command = command
            result = `#{command}`
            @last_output = result
            verify_exit_success($?, result) if verify
            return result
        end

        # Verify that CLI exit status is 0.
        # If it fails, include the command output in the error message.
        def verify_exit_success(processStatus, output)
            errorMsg = "CLI return code %d is not 0. Command Output=%s" % [processStatus.exitstatus, output]
            expect(processStatus.exitstatus).to eq(0), errorMsg
        end

        # Returns the last CLI command run.
        def last_command
            return @last_command
        end

        # Returns the last output from the last CLI command run.
        def last_output
            return @last_output
        end

        # Takes a CLI command and returns the json object for it.
        # On a json error, log the last cli run and raise the exception.
        def get_json(command)
            begin
                #puts "get_json: executing serviced #{command}"
                result = execute(command, false, false)
                result = "{}" if result.strip.length == 0
                return JSON.parse(result)
            rescue StandardError => err
                printf("Error converting response to JSON: %s\n", err)
                printf "\n=======================================================================================\n"
                printf "Last cli command:\n"
                printf "--------------------------------------------\n"
                printf "CLI Command: %s\n", CC.CLI.last_command
                printf "CLI Output:\n%s\n", CC.CLI.last_output
                printf "\n=======================================================================================\n"
                raise
            end
        end

        private

        def get_serviced_cli()
            return "/capybara/serviced --endpoint #{TARGET_HOST}:4979"
        end
    end
end
