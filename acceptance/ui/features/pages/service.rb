require_relative 'navbar'
require 'site_prism'

class Service < SitePrism::Page
    set_url applicationURL("#/services?disable-animation=true")
    set_url_matcher /services/

    section :navbar, NavBarSection, ".navbar-collapse"
    
    element :addPublicEndpoint_button, "[ng-click='modalAddPublicEndpoint()']"
    element :publicEndpoints_table, "table[data-config='publicEndpointsTable']"
    element :ipAssignments_table, "table[data-config='ipsTable']"
    element :configFiles_table, "table[data-config='configTable']"
    
end
