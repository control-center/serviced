require_relative 'navbar'
require 'site_prism'

class Applications < SitePrism::Page
    set_url applicationURL("#/apps?disable-animation=true")
    set_url_matcher /apps/

    section :navbar, NavBarSection, ".navbar-collapse"
    
    element :addApp_button, "[ng-click='modal_deployWizard()']"
    element :addAppTemplate_button, "[ng-click='modal_addTemplate()']"
    element :servicesMap_button, "a[href='/#/servicesmap'][class='btn-link']"
    element :deploymentID_field, "input[name='deploymentID']"
    element :services_table, "table[data-config='servicesTable']"
    element :templates_table, "table[data-config='templatesTable']"
    elements :status_icons, "[data-status$='service.status']"
end
