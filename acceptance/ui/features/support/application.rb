require 'uri'

#
# Returns the full application URL given a path relative to the application's base URL
#
def applicationURL(relativePath)
	return URI.join(Capybara.app_host, relativePath).to_s
end

#
# Returns the default application user id
#
def applicationUserID()
	return ENV["APPLICATION_USERID"]
end

#
# Returns the default application password
#
def applicationPassword()
	return ENV["APPLICATION_PASSWORD"]
end
