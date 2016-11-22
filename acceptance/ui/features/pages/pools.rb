require_relative 'navbar'
require 'site_prism'

class Pools < SitePrism::Page
    set_url applicationURL("#/pools?disable-animation=true&loglevel=debug")
    set_url_matcher /pools/

    section :navbar, NavBarSection, ".navbar-collapse"

    element :addPool_button, ".add-pool-button"
    element :poolName_input, "input[name='new_pool_name']"
    element :description_input, "input[name='new_pool_description']"
    element :addVirtualIp_button, "[ng-click='modalAddVirtualIp(currentPool)']"
    element :ip_input, "input[name='ip']"
    element :netmask_input, "input[name='netmask']"
    element :interface_input, "input[name='interface']"
    element :dialogAddVirtualIp_button, "button[class='btn btn-primary submit']"
    element :virtualIps_table, "table[data-config='virtualIPsTable']"
    elements :pool_names, "td[data-title-text='Resource Pool']"
end
