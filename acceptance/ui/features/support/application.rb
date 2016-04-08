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

#
# Retries a method while catching errors.
#
# Example: retryMethod( method( :yourFunction ), 1, 5 )
#
#    Calls yourFunction().  If it encounters an error it
#    will wait 5 seconds and retry once.
#
def retryMethod(function, retries, delay)
    for i in (1..retries)
        begin
            # Make an attempt at the method call, catching errors.
            function.call()
            return # Success.
        rescue StandardError => e
            printf("retryMethod: %s\n" % [e.message])
            printf("** Sleeping %d seconds before retrying.\n" % [delay])
            sleep delay
        end
    end

    # Allow this to throw an error.
    function.call()
end

