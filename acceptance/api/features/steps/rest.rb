When(/^I send a (GET|POST|PATCH|PUT|DELETE) payload request to CC at "(.*?)" with body "(.*?)"$/) do |method, path, body|
  make_request(method, path, body)
end

When(/^I send a (GET|POST|PATCH|PUT|DELETE) request to CC at "(.*?)"$/) do |method, path|
  b = nil
  make_request(method, path, b)
end

When(/^I send a (GET|POST|PATCH|PUT|DELETE) request from file "(.*?)" to CC at "(.*?)"$/) do |method, filename, path|
  contents = File.read(File.join(ENV["DATASET_DIR"], filename))
  make_request(method, path, JSON.parse(contents))
end

def make_request(method, path, body)
    hosturl = ENV["APPLICATION_URL"]
    url = %$#{hosturl}#{path}$
    request_url = URI.encode resolve(url)
    @body = body
    if 'GET' == %/#{method}/ and $cache.has_key? %/#{request_url}/
        @response = $cache[%/#{request_url}/]
        @headers = nil
        @body = nil
        @grabbed = nil
        return
    end

    @headers = {} if @headers.nil?
    bodyJson = @body.nil? ? "" : @body.to_json

    begin
        case method
            when 'GET'
                response = RestClient::Request.execute(:method => :get, :url => request_url, :headers => @headers, :verify_ssl => false, :cookies => @cookies)
            when 'POST'
                response = RestClient::Request.execute(:method => :post, :url => request_url, :headers => @headers, :payload => bodyJson, :verify_ssl => false, :cookies => @cookies)
            when 'PATCH'
                response = RestClient.patch request_url, @body, @headers
            when 'PUT'
                response = RestClient::Request.execute(:method => :put, :url => request_url, :headers => @headers, :payload => bodyJson, :verify_ssl => false, :cookies => @cookies)
            else
                response = RestClient::Request.execute(:method => :delete, :url => request_url, :headers => @headers, :payload => bodyJson, :verify_ssl => false, :cookies => @cookies)
        end
    rescue RestClient::Exception => e
        response = e.response
    end

    @response = CucumberApi::Response.create response
    @headers = nil
    @body = nil
    @grabbed = nil
    $cache[%/#{request_url}/] = @response if 'GET' == %/#{method}/
end

# find a value in the response
Then(/^the JSON response should have value "(.*?)" at "(.*?)"$/) do |value, jsonpath|
  data = @response.get jsonpath
  if data == nil or !data.to_s.include? value
    raise "Could not find #{value} at path #{jsonpath}, data was: #{data}, body: #{@response.body}, code: #{@response.code}"
  end
end

