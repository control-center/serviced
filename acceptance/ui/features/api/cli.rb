require "json"
require "capybara/rspec"

#
# Encapsulates the CLI functionality to be exposed to the
# acceptance tests: CC.CLI.<function>
#
class CLI
    include ::RSpec::Matchers

    def initialize()
        @last_command = ""
        @last_output = ""
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
            result = execute(command, false)
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

    # Ensures that only the host provided is there.  If not, remove them and
    # add the default host.
    def ensure_only_default_host_exists()
        host = getTableValue("table://hosts/defaultHost/name")
        json = get_json("%{serviced} host list -v")
        if (json.length != 1 || findArrayMatch(json, "Name", host) == nil)
            remove_all_hosts()
            add_default_host()
        end
    end

    def remove_all_hosts()
        result = execute("%{serviced} host list --show-fields ID 2>&1 | grep -v ^ID | xargs --no-run-if-empty %{serviced} host rm")

        # verify all of the hosts were really removed
        result = execute("%{serviced} host list")
        expect(result).to include("no hosts found")
    end

    def add_host(name, port, pool, commitment, hostID)
        nameValue =  getTableValue(name)
        portValue =  getTableValue(port)
        poolValue =  getTableValue(pool)
        commitmentValue =  getTableValue(commitment)

        result = execute("%{serviced} host add '#{nameValue}:#{portValue}' '#{poolValue}' --memory '#{commitmentValue}'")

        hostIDValue =  getTableValue(hostID)
        expect(result.strip).to eq(hostIDValue.to_s)
    end

    # Verify that the port properties exist in the port list output.
    def verify_publicendpoint_port_list_matches(name)
        service = getTableValue("table://ports/#{name}/Service")
        endpoint = getTableValue("table://ports/#{name}/Endpoint")
        portAddr = getTableValue("table://ports/#{name}/PortAddr")
        protocol = getTableValue("table://ports/#{name}/Protocol")
        enabled = getTableValue("table://ports/#{name}/Enabled")

        # Make sure one of the lines matches each of the values
        result = execute("%{serviced} service public-endpoints port list")
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
        result = execute("%{serviced} service public-endpoints vhost list")
        result.each_line do |line|
            return true if (result =~ /#{service}/) && (result =~ /#{endpoint}/) && (result =~ /#{portAddr}/) && (result =~ /#{protocol}/) && (result =~ /#{enabled}/)
        end

        fail(ArgumentError.new("vhost #{name} doesn't exist"))
    end

    # Verify that CLI exit status is 0.
    # If it fails, include the command output in the error message.
    def verify_exit_success(processStatus, output)
        errorMsg = "CLI return code %d is not 0. Command Output=%s" % [processStatus.exitstatus, output]
        expect(processStatus.exitstatus).to eq(0), errorMsg
    end

    def check_service_exists(serviceName)
        serviceName = getTableValue(serviceName)
        result = execute("%{serviced} service list --show-fields Name")
        matchData = result.match /^#{serviceName}$/
        return matchData != nil
    end

    # Checks the CLI output to see if an app/id exists.
    def check_service_with_id_exists(app, id)
        app = getTableValue(app)
        id = getTableValue(id)
        json = get_json("%{serviced} service list -v")
        json.each do |service|
            return true if service["Name"] == app && service["DeploymentID"] == id
        end
        return nil
    end

    # Deploys the service.
    def add_service(template, pool, id)
        template = getTableValue(template)
        templateID = get_template_id(template)
        pool = getTableValue(pool)
        id = getTableValue(id)
        execute("%{serviced} template deploy #{templateID} #{pool} #{id}")
        check_service_with_id_exists(template, id)
    end

    def remove_service(serviceName)
        serviceName = getTableValue(serviceName)
        execute("%{serviced} service rm #{serviceName}")
    end

    def remove_all_services()
        execute("%{serviced} service list --show-fields ServiceID 2>&1 | grep -v ServiceID | xargs --no-run-if-empty %{serviced} service rm")
    end

    def add_template(dir)
        templateID = execute("%{serviced} template compile #{dir} | %{serviced} template add")
        result = execute("%{serviced} template list #{templateID}")
        expect(result.lines.count).not_to eq(0)
    end

    def get_template_id(templateName)
        templateName = getTableValue(templateName)
        json = get_json("%{serviced} template list -v")
        template = findArrayMatch(json, "Name", templateName)
        id = template["ID"]
        expect(id).not_to eq(nil)
        return id
    end

    def check_template_exists?(templateName)
        templateName = getTableValue(templateName)
        result = execute("%{serviced} template list --show-fields Name")
        matchData = result.match /^#{templateName}$/
        return matchData != nil
    end

    def remove_all_templates()
        execute("%{serviced} template list --show-fields TemplateID 2>&1 | grep -v TemplateID | xargs --no-run-if-empty %{serviced} template rm")
    end

    def add_pool(name, description)
        # Description isn't used in the CLI.
        nameValue =  getTableValue(name)
        result = execute("%{serviced} pool add '#{nameValue}'")
        expect(result.strip).to eq(nameValue.to_s)
    end

    def add_default_pool()
        add_pool_json("defaultPool")
    end

    def add_pool_json(pool)
        add_pool("table://pools/" + pool + "/name", "table://pools/" + pool + "/description")
    end

    def check_pool_exists(poolName)
        result = execute("%{serviced} pool list")
        matchData = result.match /^#{poolName}$/
        return matchData != nil
    end

    def remove_all_pools_except_default()
        remove_all_services()
        remove_all_hosts()
        remove_all_resource_pools_except_default()
    end

    def remove_all_resource_pools_except_default()
        execute("%{serviced} pool list --show-fields ID 2>&1 | grep -v ^ID | grep -v ^default | xargs --no-run-if-empty %{serviced} pool rm")

        # verify all of the hosts were really removed
        result = execute("%{serviced} pool list --show-fields ID 2>&1 | grep -v ^ID", false, true)
        expect(result.strip).to eq("default")
    end

    def remove_virtual_ips_from_default_pool()
        execute("%{serviced} pool list-ips --show-fields IPAddress default 2>/dev/null | grep -v ^IPAddress | xargs --no-run-if-empty %{serviced} pool remove-virtual-ip default")
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

    private

    def get_serviced_cli()
        return "/capybara/serviced --endpoint #{TARGET_HOST}:4979"
    end

    def add_publicendpoint_port(service, endpoint, portAddr, protocol, enabled)
        execute("%{serviced} service public-endpoints port add #{service} #{endpoint} #{portAddr} #{protocol} #{enabled}")
    end

    # Returns the matching port definition from the service or nil
    def check_publicendpoint_port_exists(service, endpoint, portAddr)
        result = get_json("%{serviced} service list #{service} -v")

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
        execute("%{serviced} service public-endpoints port rm #{service} #{endpoint} #{portAddr}")
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

