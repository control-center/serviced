case node["platform_family"]
when "debian"

  default['docker']['package']['distribution'] = "docker"
  default['docker']['package']['repo_url'] = "https://get.docker.io/ubuntu"
  default['docker']['package']['repo_key'] = "https://get.docker.io/gpg"

when "fedora"
  
  default['docker']['package']['repo_url'] = "http://goldmann.fedorapeople.org/repos/docker.repo"
  default['docker']['package']['name'] = "docker-io"
  
when "rhel"

  

when "gentoo"

  

end