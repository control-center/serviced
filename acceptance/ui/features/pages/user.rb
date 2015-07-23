require_relative 'navbar'
require 'site_prism'

class User < SitePrism::Page
    section :navbar, NavBarSection, ".navbar-collapse"

    element :clearMessages_button, "[ng-click='clearMessages()']"
    element :unreadMessage, "[class='message unreadMessage ng-scope']"
    element :english_button, "label[class^='btn']", :text => 'English'
    element :spanish_button, "label[class^='btn']", :text => 'Esp'
    element :message, "[class^='message ']"
end
