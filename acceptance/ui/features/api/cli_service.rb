module CCApi
    #
    # Encapsulates the 'serviced service..' functionality to be exposed to the
    # acceptance tests: CC.CLI.service.<function>
    #
    class CLI_Service
        # Verify that the port properties exist in the port list output.
        def verify_publicendpoint_port_list_matches(name)
            service = getTableValue("table://ports/#{name}/Service")
            endpoint = getTableValue("table://ports/#{name}/Endpoint")
            portAddr = getTableValue("table://ports/#{name}/PortAddr")
            protocol = getTableValue("table://ports/#{name}/Protocol")
            enabled = getTableValue("table://ports/#{name}/Enabled")

            # Make sure one of the lines matches each of the values
            result = CC.CLI.execute("%{serviced} service public-endpoints port list")
            result.each_line do |line|
                return true if (result =~ /#{service}/) && (result =~ /#{endpoint}/) && (result =~ /#{portAddr}/) && (result =~ /#{protocol}/) && (result =~ /#{enabled}/)
            end

            fail(ArgumentError.new("port #{name} doesn't exist"))
        end

        # Adds the port public endpoint from the table definition, using the cli
        def add_publicendpoint_port_json(name)
            service = getTableValue("table://ports/#{name}/Service")
            endpoint = getTableValue("table://ports/#{name}/Endpoint")
            portAddr = getTableValue("table://ports/#{name}/PortAddr")
            protocol = getTableValue("table://ports/#{name}/Protocol")
            enabled = getTableValue("table://ports/#{name}/Enabled")
            add_publicendpoint_port(service, endpoint, portAddr, protocol, enabled)
        end

        def check_publicendpoint_port_exists_json(name)
            service = getTableValue("table://ports/#{name}/Service")
            endpoint = getTableValue("table://ports/#{name}/Endpoint")
            portAddr = getTableValue("table://ports/#{name}/PortAddr")
            return check_publicendpoint_port_exists(service, endpoint, portAddr)
        end

        def remove_publicendpoint_port_json(name)
            service = getTableValue("table://ports/#{name}/Service")
            endpoint = getTableValue("table://ports/#{name}/Endpoint")
            portAddr = getTableValue("table://ports/#{name}/PortAddr")
            remove_publicendpoint_port(service, endpoint, portAddr)
        end

        # Verify that the vhost properties exist in the port list output.
        def check_publicendpoint_vhost_exists_json(name)
            service = getTableValue("table://vhosts/#{name}/Service")
            endpoint = getTableValue("table://vhosts/#{name}/Endpoint")
            portAddr = getTableValue("table://vhosts/#{name}/PortAddr")
            protocol = getTableValue("table://vhosts/#{name}/Protocol")
            enabled = getTableValue("table://vhosts/#{name}/Enabled")
            return check_publicendpoint_vhost_exists(service, endpoint, portAddr, protocol, enabled)
        end

        def check_publicendpoint_vhost_exists(service, endpoint, portAddr, protocol, enabled)
            # Make sure one of the lines matches each of the values
            result = CC.CLI.execute("%{serviced} service public-endpoints vhost list")
            result.each_line do |line|
                return true if (result =~ /#{service}/) && (result =~ /#{endpoint}/) && (result =~ /#{portAddr}/) && (result =~ /#{protocol}/) && (result =~ /#{enabled}/)
            end

            fail(ArgumentError.new("vhost #{name} doesn't exist"))
        end

        def check_service_exists(serviceName)
            serviceName = getTableValue(serviceName)
            result = CC.CLI.execute("%{serviced} service list --show-fields Name")
            matchData = result.match /^#{serviceName}$/
            return matchData != nil
        end

        # Checks the CLI output to see if an app/id exists.
        def service_with_id_exists?(app, id)
            app = getTableValue(app)
            id = getTableValue(id)
            json = CC.CLI.get_json("%{serviced} service list -v")
            json.each do |service|
                return true if service["Name"] == app && service["DeploymentID"] == id
            end
            return false
        end

        def remove_service(serviceName)
            serviceName = getTableValue(serviceName)
            CC.CLI.execute("%{serviced} service rm #{serviceName}")
        end

        def remove_all_services()
            CC.CLI.execute("%{serviced} service list --show-fields ServiceID 2>&1 | grep -v ServiceID | xargs --no-run-if-empty %{serviced} service rm")
        end

        # Looks up the given port from the ports table and enables/disables it.
        def enable_publicendpoint_port_json(name, enabled)
            service = getTableValue("table://ports/#{name}/Service")
            endpoint = getTableValue("table://ports/#{name}/Endpoint")
            portAddr = getTableValue("table://ports/#{name}/PortAddr")
            return enable_publicendpoint_port(service, endpoint, portAddr, enabled)
        end

        # Enables/disables the given port.
        def enable_publicendpoint_port(service, endpoint, portAddr, enabled)
            CC.CLI.execute("%{serviced} service public-endpoints port enable #{service} #{endpoint} #{portAddr} #{enabled}")
        end

        # Looks up the given port from the ports table, and returns the enabled
        # state based on the service definition.
        def check_publicendpoint_port_enabled_json?(name)
            service = getTableValue("table://ports/#{name}/Service")
            endpoint = getTableValue("table://ports/#{name}/Endpoint")
            portAddr = getTableValue("table://ports/#{name}/PortAddr")
            table_protocol = getTableValue("table://ports/#{name}/Protocol")
            protocol = map_protocol_value(table_protocol)
            usetls = map_tls_value(table_protocol)
            enabled = getTableValue("table://ports/#{name}/Enabled")
            return check_publicendpoint_port_exists_in_service?(service, endpoint, portAddr, protocol, usetls, enabled)
        end

        # returns the enabled state of a port based on the service definition.
        def check_publicendpoint_port_exists_in_service?(service, endpoint, portAddr, protocol, usetls, enabled)
            json = CC.CLI.get_json("%{serviced} service list #{service}")
            endpoint = findArrayMatch(json["Endpoints"], "Name", endpoint)
            fail(ArgumentError.new("endpoint #{endpoint} doesn't exist in service #{service}")) if endpoint == nil
            port = findArrayMatch(endpoint["PortList"], "PortAddr", portAddr)
            fail(ArgumentError.new("port #{portAddr} doesn't exist in endpoint #{endpoint}")) if port == nil
            return true if port["Protocol"] == protocol and port["UseTLS"] == usetls and port["Enabled"] == enabled
            return false
        end

        private

        def add_publicendpoint_port(service, endpoint, portAddr, protocol, enabled)
            CC.CLI.execute("%{serviced} service public-endpoints port add #{service} #{endpoint} #{portAddr} #{protocol} #{enabled}")
        end

        # Returns the matching port definition from the service or nil
        def check_publicendpoint_port_exists(service, endpoint, portAddr)
            result = CC.CLI.get_json("%{serviced} service list #{service} -v")

            endPoints = result["Endpoints"]
            return nil if endPoints == nil
            endpoint = findArrayMatch(endPoints, "Name", endpoint)
            return nil if endpoint == nil
            portList = endpoint["PortList"]
            return nil if portList == nil

            # Make sure we have an endpoint that matches the port address.
            return findArrayMatch(portList, "PortAddr", portAddr)
        end

        def remove_publicendpoint_port(service, endpoint, portAddr)
            CC.CLI.execute("%{serviced} service public-endpoints port rm #{service} #{endpoint} #{portAddr}")
        end

        # Maps the ui/list/table value for the protocol to the stored service def value.
        def map_protocol_value(protocol)
            protocol = protocol.downcase
            case protocol
            when "other"
                return ""
            when "other-tls"
                return ""
            end
            return protocol
        end

        # Maps the ui/list value for the protocol to the tls value.
        def map_tls_value(protocol)
            case protocol.downcase
            when "https"
                return true
            when "other-tls"
                return true
            end
            return false
        end
    end
end