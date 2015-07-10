require_relative 'navbar'
require 'site_prism'

class ServicesMap < SitePrism::Page
    set_url applicationURL("#/servicesmap")
    set_url_matcher /servicesmap/

    section :navbar, NavBarSection, ".navbar-collapse"
    
    element :map, "svg[class='tall']"
end