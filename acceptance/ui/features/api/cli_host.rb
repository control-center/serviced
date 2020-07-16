module CCApi
    #
    # Encapsulates the 'serviced host..' functionality to be exposed to the
    # acceptance tests: CC.CLI.host.<function>
    #
    class CLI_Host
        include ::RSpec::Matchers

        # Ensures that only the host provided is there.  If not, remove them and
        # add the default host.
        def ensure_only_default_host_exists()
            host = getTableValue("table://hosts/defaultHost/name")
            json = CC.CLI.get_json("%{serviced} host list -v")
            if (json.length != 1 || findArrayMatch(json, "Name", host) == nil)
                remove_all_hosts()
                add_default_host()
            end
        end

        def remove_all_hosts()
            result = CC.CLI.execute(
                "%{serviced} host list --show-fields ID 2>&1 | grep -v ^ID | xargs --no-run-if-empty %{serviced} host rm",
                stderr = true,
                verify = false
            )

            # verify all of the hosts were really removed
            result = CC.CLI.execute("%{serviced} host list")
            expect(result).to include("no hosts found")
        end

        def add_host(name, port, pool, commitment, hostID)
            nameValue =  getTableValue(name)
            portValue =  getTableValue(port)
            poolValue =  getTableValue(pool)
            commitmentValue =  getTableValue(commitment)

            result = CC.CLI.execute("%{serviced} host add '#{nameValue}:#{portValue}' '#{poolValue}' --memory '#{commitmentValue}' -k /dev/null")
            result = result.split("\n")[-1]

            hostIDValue =  getTableValue(hostID)
            expect(result.strip).to eq(hostIDValue.to_s)
        end

        def add_host_json(host)
            name = "table://hosts/" + host + "/hostName"
            port = "table://hosts/" + host + "/rpcPort"
            pool = "table://hosts/" + host + "/pool"
            commitment = "table://hosts/" + host + "/commitment"
            hostID = "table://hosts/" + host + "/hostID"

            add_host(name, port, pool, commitment, hostID)
        end

        def add_default_host()
            add_host_json("defaultHost")
        end
    end
end
