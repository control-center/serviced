require 'site_prism'

class Applications < SitePrism::Page
  set_url applicationURL("#/apps")
  set_url_matcher /apps/
end
