require 'site_prism'

class Hosts < SitePrism::Page
  set_url applicationURL("#/hosts")
  set_url_matcher /hosts/

  element :addHost_button, "[ng-click='modalAddHost()']"
  element :hostsMap_button, "a[href='/#/hostsmap][class='btn-link]"
  element :hostName_input, "#new_host_name"
  element :resourcePool_input, "[name='new_host_parent']"
  element :ramCommitment_input, "[name='new_host_ram_commitment']"
end
