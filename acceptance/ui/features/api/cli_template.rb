module CCApi
    #
    # Encapsulates the 'serviced template..' functionality to be exposed to the
    # acceptance tests: CC.CLI.template.<function>
    #
    class CLI_Template
        include ::RSpec::Matchers

        # Deploys the service.
        def deploy(template, pool, id)
            template = getTableValue(template)
            templateID = get_template_id(template)
            pool = getTableValue(pool)
            id = getTableValue(id)
            CC.CLI.execute("%{serviced} template deploy #{templateID} #{pool} #{id}")
            expect(CC.CLI.service.service_with_id_exists?(template, id))
        end

        def add_template(dir)
            templateID = CC.CLI.execute("%{serviced} template compile #{dir} | %{serviced} template add")
            result = CC.CLI.execute("%{serviced} template list #{templateID}")
            expect(result.lines.count).not_to eq(0)
        end

        def get_template_id(templateName)
            templateName = getTableValue(templateName)
            json = CC.CLI.get_json("%{serviced} template list -v")
            template = findArrayMatch(json, "Name", templateName)
            id = template["ID"]
            expect(id).not_to eq(nil)
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