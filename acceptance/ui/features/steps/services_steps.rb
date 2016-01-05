Then (/^VHost "(.*?)" should exist$/) do |vhost|
    expect(checkVHostExists(vhost))
end

Then (/^Public Port "(.*?)" should exist$/) do |port|
    expect(checkPublicPortExists(port))
end

def checkVHostExists(vhost)
    initServicePageObj()
    vhostName = getTableValue(vhost)
    searchStr = "https://#{vhostName}."

    found = false
    within(@service_page.publicEndpoints_table) do
        found = page.has_text?(searchStr)
    end
    return found
end

def checkPublicPortExists(port)
    initServicePageObj()
    portName = getTableValue(port)
    searchStr = ":#{portName}"

    found = false
    within(@service_page.publicEndpoints_table) do
        found = page.has_text?(searchStr)
    end
    return found
end

def initServicePageObj()
    if @service_page.nil?
        @service_page = Service.new
    end
end
