Before('@login-required') do
  @cookies = nil  
  host = ENV["APPLICATION_URL"]
  url = %$https://#{host}/login$
  user = ENV["APPLICATION_USERID"]
  pass = ENV["APPLICATION_PASSWORD"]
  payload = %/{"Username":"#{user}", "Password":"#{pass}"}/
  response = RestClient::Request.execute(:method => :post, :url => url, :payload => payload, :verify_ssl => false, :content_type => 'application/json' )
  @cookies = response.cookies
end
