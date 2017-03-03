require_relative 'navbar'
require 'site_prism'

class Hosts < SitePrism::Page
    set_url applicationURL("#/hosts?disable-animation=true&loglevel=debug")
    set_url_matcher /hosts/

    section :navbar, NavBarSection, ".navbar-collapse"

    element :addHost_button, ".add-host-button"
    element :hostHost_input, "#new_host_name"
    element :rpcPort_input, "#new_host_port"
    element :resourcePool_input, "[name='new_host_parent']"
    element :ramLimit_input, "[name='new_host_ram_commitment']"
    element :addHost_dialog, ".modal-content"
    element :host_entry, "[ng-repeat='host in $data']"
    element :invalidPort_input, "[class$='ng-invalid ng-invalid-pattern']#new_host_port"
    elements :active_icons, "[ng-if$='host.active']"
    elements :host_entries, "[ng-repeat='host in $data']"

    # type selector
    element :checkNAT, :xpath, '//*[@id="ckbox_use_nat"]'

    # nat inputs
    element :natHost_input, "#new_host_nat_host"
    element :natPort_input, "#new_host_nat_port"
end
