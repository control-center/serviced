require 'site_prism'

class NavBarSection < SitePrism::Section
    element :applications, "a[href='#/apps']"
    element :resourcePools, "a[href='#/pools']"
    element :hosts, "a[href='#/hosts']"
    element :logs, "a[href='#/logs']"
    element :backup, "a[href='#/backuprestore']"
    element :userDetails, "button[ng-click='modalUserDetails()']"
    element :help, "a[href='/static/doc/']"
    element :logout, "button[ng-click='logout()']"
    element :about, "button[ng-click='modalAbout()']"
end