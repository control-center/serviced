require_relative 'navbar'
require 'site_prism'

class User < SitePrism::Page
    section :navbar, NavBarSection, ".navbar-collapse"

    element :clearMessages_button, "[ng-click='clearMessages()']"
    element :unreadMessage, "[class='message unreadMessage ng-scope']"
    element :english_button, "input[value='en_US']"
    element :spanish_button, "input[value='es_US']"
    element :message, "[class^='message ']"
end
