Before('@login-required') do
  @cookies = nil  
  url = ENV["APPLICATION_URL"]
  loginurl = %$#{url}/login$
  user = ENV["APPLICATION_USERID"]
  pass = ENV["APPLICATION_PASSWORD"]
  payload = %/{"Username":"#{user}", "Password":"#{pass}"}/
  response = RestClient::Request.execute(:method => :post, :url => loginurl, :payload => payload, :verify_ssl => false, :content_type => 'application/json' )
  @cookies = response.cookies
end

Before('@reload_service') do
  CC.CLI.service.clean_remove('testsvc', 2)
end
