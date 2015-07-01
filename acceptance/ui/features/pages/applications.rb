require 'site_prism'

class Applications < SitePrism::Page
    set_url applicationURL("#/apps")
    set_url_matcher /apps/

    element :addApp_button, "[ng-click='modal_deployWizard()']"
    element :addAppTemplate_button, "[ng-click='modal_addTemplate()']"
    element :servicesMap_button, "a[href='/#/servicesmap'][class='btn-link']"
    element :deploymentID_field, "input[name='deploymentID']"
end