require_relative 'navbar'
require 'site_prism'

class Service < SitePrism::Page
    set_url applicationURL("#/services?disable-animation=true&loglevel=debug&no-focusme=true")
    set_url_matcher /services/

    section :navbar, NavBarSection, ".navbar-collapse"

    element :addPublicEndpoint_button, "[ng-click='modalAddPublicEndpoint()']"
    element :publicEndpoints_table, "table[data-config='publicEndpointsTable']"
    element :ipAssignments_table,   "table[data-config='ipsTable']"
    element :configFiles_table,     "table[data-config='configTable']"

    # add public endpoint dialog elements
    element :addPublicEndpoint_dialog, ".modal-content"
    element :addVHostApp_select,    :xpath, "//select[@id='add_vhost_application']"

    # type selector
    element :buttonPortType,        "div.btn-group", "//input[@value='port']"
    element :buttonVHostType,       "div.btn-group", "//input[@value='vhost']"

    # type port
    element :newHostName_input,     :xpath, "//input[@name='new_host_name']"
    element :newPort_input,         :xpath, "//input[@id='add_public_endpoint_port']"
    element :addProtocol_select,    :xpath, "//select[@id='add_port_protocol']"

    # type vhost
    element :newVHost_input,        :xpath, "//input[@id='add_vhost_vhost']"

end
