require_relative 'navbar'
require 'site_prism'

class User < SitePrism::Page
    section :navbar, NavBarSection, ".navbar-collapse"
end
