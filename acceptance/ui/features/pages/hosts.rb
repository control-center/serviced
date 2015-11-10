require_relative 'navbar'
require 'site_prism'

class Hosts < SitePrism::Page
    set_url applicationURL("#/hosts?disable-animation=true")
    set_url_matcher /hosts/

    section :navbar, NavBarSection, ".navbar-collapse"

    element :addHost_button, "[ng-click='modalAddHost()']"
    element :hostsMap_button, "a[href='/#/hostsmap'][class='btn-link']"
    element :hostName_input, "#new_host_name"
    element :resourcePool_input, "[name='new_host_parent']"
    element :ramCommitment_input, "[name='new_host_ram_commitment']"
    element :addHost_dialog, ".modal-content"
    element :host_entry, "[ng-repeat='host in $data']"
    element :invalidHost_input, "[class$='ng-invalid ng-invalid-pattern']#new_host_name"
    elements :active_icons, "[ng-if$='host.active']"
    elements :host_entries, "[ng-repeat='host in $data']"
end
