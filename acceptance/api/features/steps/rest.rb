
When(/^I send a (GET|POST|PATCH|PUT|DELETE) request to CC at "(.*?)"$/) do |method, url|
  request_url = URI.encode resolve(url)
  if 'GET' == %/#{method}/ and $cache.has_key? %/#{request_url}/
    @response = $cache[%/#{request_url}/]
    @headers = nil
    @body = nil
    @grabbed = nil
    next
  end

  @headers = {} if @headers.nil?
  bodyJson = @body.nil? ?  "" : @body.to_json
  begin
    case method
    when 'GET'
      response = RestClient::Request.execute(:method => :get, :url => request_url, :headers => @headers, :verify_ssl => false , :cookies => @cookies)  
    when 'POST'
      response = RestClient::Request.execute(:method => :post, :url => request_url, :headers => @headers, :payload => bodyJson, :verify_ssl => false, :cookies => @cookies)  
    when 'PATCH'
      response = RestClient.patch request_url, @body, @headers
    when 'PUT'
      response = RestClient.put request_url, @body, @headers
    else
      response = RestClient.delete request_url, @headers
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
