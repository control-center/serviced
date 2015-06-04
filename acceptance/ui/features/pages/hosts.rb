require 'site_prism'

class Hosts < SitePrism::Page
  set_url applicationURL("#/hosts")
  set_url_matcher /hosts/

  element :addHosts_button, "[ng-click='modalAddHost()']"
  element :hostName_input, "#new_host_name"
  element :resourcePool_input, "[name='new_host_parent']"
  element :ramCommitment_input, "[name='new_host_ram_commitment']"
end
