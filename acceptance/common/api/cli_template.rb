module CCApi
    #
    # Encapsulates the 'serviced template..' functionality to be exposed to the
    # acceptance tests: CC.CLI.template.<function>
    #
    class CLI_Template

        # Deploys the service.
        def deploy(template, pool, id)
            if CC.CLI.service.service_with_id_exists?(template, id)
                return
            end
            template = getTableValue(template)
            templateID = get_template_id(template)
            pool = getTableValue(pool)
            id = getTableValue(id)
            CC.CLI.execute("%{serviced} template deploy #{templateID} #{pool} #{id}")
            if !CC.CLI.service.service_with_id_exists?(template, id)
                raise "Failed to deploy the service!"
            end
        end

        def add_template(dir)
            templateID = CC.CLI.execute("%{serviced} template compile #{dir} | %{serviced} template add")
            result = CC.CLI.execute("%{serviced} template list #{templateID}")
            if result.lines.count == 0
                raise "Error adding template!: #{result}"
            end
        end

        def get_template_id(templateName)
            templateName = getTableValue(templateName)
            json = CC.CLI.get_json("%{serviced} template list -v")
            template = findArrayMatch(json, "Name", templateName)
            id = template["ID"]
            if id == nil
                raise "Failed to get template ID!"
            end
            return id
        end

        def check_template_exists?(templateName)
            templateName = getTableValue(templateName)
            result = CC.CLI.execute("%{serviced} template list --show-fields Name")
            matchData = result.match /^#{templateName}$/
            return matchData != nil
        end

        def remove_all_templates()
            CC.CLI.execute(
                "%{serviced} template list --show-fields TemplateID 2>&1 | grep -v TemplateID | xargs --no-run-if-empty %{serviced} template rm",
                stderr = true,
                verify = false
            )
        end
    end
end