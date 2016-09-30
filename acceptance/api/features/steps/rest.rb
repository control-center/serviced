When(/^I send a (GET|POST|PATCH|PUT|DELETE) payload request to CC at "(.*?)" with body "(.*?)"$/) do |method, path, body|

  host = ENV["APPLICATION_URL"]
  url = %$https://#{host}#{path}$
  request_url = URI.encode resolve(url)
  @body = body
  if 'GET' == %/#{method}/ and $cache.has_key? %/#{request_url}/
    @response = $cache[%/#{request_url}/]
    @headers = nil
    @body = nil
    @grabbed = nil
    next
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
        response = RestClient.Request.execute(:method => :put, :url => request_url, :headers => @headers, :payload => bodyJson, :verify_ssl => false, :cookies => @cookies)
      else
        response = RestClient.Request.execute(:method => :delete, :url => request_url, :headers => @headers, :payload => bodyJson, :verify_ssl => false, :cookies => @cookies)
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

When(/^I send a (GET|POST|PATCH|PUT|DELETE) request to CC at "(.*?)"$/) do |method, path|
  b = nil
  steps %Q{
    When I send a #{method} payload request to CC at "#{path}" with body "#{b}"
  }
end


# find a value in the response
Then(/^the JSON response body should have value "(.*?)" at jsonpath "(.*?)"$/) do |value, jsonpath|
  data = @response.get jsonpath
  if data == nil or data != value
    raise "Could not find #{value} at the specified path #{jsonpath}"
  end
end

