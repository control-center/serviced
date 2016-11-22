require_relative './CC'

module CCApi
    #
    # Encapsulates the 'serviced pool..' functionality to be exposed to the
    # acceptance tests: CC.CLI.pool.<function>
    #
    class CLI_Pool
        include ::RSpec::Matchers

        def add_pool(name, description)
            # Description isn't used in the CLI.
            nameValue =  getTableValue(name)
            result = CC.CLI.execute("%{serviced} pool add '#{nameValue}'")
            expect(result.strip).to eq(nameValue.to_s)
        end

        def add_default_pool()
            add_pool_json("defaultPool")
        end

        def add_pool_json(pool)
            add_pool("table://pools/" + pool + "/name", "table://pools/" + pool + "/description")
        end

        def check_pool_exists(poolName)
            result = CC.CLI.execute("%{serviced} pool list --show-fields ID")
            matchData = result.match /^#{poolName}$/
            return matchData != nil
        end

        def remove_all_resource_pools_except_default()
            CC.CLI.execute("%{serviced} pool list --show-fields ID 2>&1 | grep -v ^ID | grep -v ^default | xargs --no-run-if-empty %{serviced} pool rm")

            # verify all of the hosts were really removed
            result = CC.CLI.execute("%{serviced} pool list --show-fields ID 2>&1 | grep -v ^ID", false, true)
            expect(result.strip).to eq("default")
        end

        def remove_virtual_ips_from_default_pool()
            CC.CLI.execute("%{serviced} pool list-ips --show-fields IPAddress default 2>/dev/null | grep -v ^IPAddress | xargs --no-run-if-empty %{serviced} pool remove-virtual-ip default")
        end
    end
end
