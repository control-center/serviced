require_relative 'navbar'
require 'site_prism'

class Pools < SitePrism::Page
    set_url applicationURL("#/pools?disable-animation=true")
    set_url_matcher /pools/

    section :navbar, NavBarSection, ".navbar-collapse"

    element :addPool_button, "[ng-click='modalAddPool()']"
    element :poolName_input, "input[name='new_pool_name']"
    element :description_input, "input[name='new_pool_description']"
    element :addVirtualIp_button, "[ng-click='modalAddVirtualIp(currentPool)']"
    element :ip_input, "[ng-model='add_virtual_ip.IP']"
    element :netmask_input, "[ng-model='add_virtual_ip.Netmask']"
    element :interface_input, "[ng-model='add_virtual_ip.BindInterface']"
    element :dialogAddVirtualIp_button, "button[class='btn btn-primary submit']"
    element :virtualIps_table, "table[data-config='virtualIPsTable']"
    elements :pool_names, "td[data-title-text='Resource Pool']"
end
