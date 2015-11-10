require 'site_prism'

class ErrorMessage < SitePrism::Section
    element :text, ".message"
end

class Login < SitePrism::Page
    set_url applicationURL("#/login?disable-animation=true")
    set_url_matcher /login/

    section :error_message, ErrorMessage, ".notification.bg-danger"

    # element :userid_field, :xpath, "//input[@ng-model='username']"
    element :userid_field, "[ng-model='username']"
    element :password_field, "[ng-model='password']"
    element :signin_button, "[type='submit']"
end
