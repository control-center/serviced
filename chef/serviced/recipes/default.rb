Chef::Log.info "Preparing to install docker for: " + node["platform_family"] + " (" + node['platform'] + ")"

case node["platform_family"]
when "debian"

  apt_repository "docker" do
    uri node['docker']['package']['repo_url']
    distribution node['docker']['package']['distribution']
    components [ "main" ]
    key node['docker']['package']['repo_key']
  end

  package "lxc-docker"
  package "linux-image-generic-lts-raring"
  package "linux-headers-generic-lts-raring"

  execute "ip_forward" do
    creates "/etc/sysctl.d/80-docker.conf"
    command "sysctl -w net.ipv4.ip_forward=1"
    action :run
  end
  
  template "/etc/sysctl.d/80-docker.conf" do
    source "80-docker.erb"
    variables()
  end
  
when "fedora"

  execute "ip_forward" do
    creates "/etc/sysctl.d/80-docker.conf"
    command "sysctl -w net.ipv4.ip_forward=1"
    action :run
  end
  
  remote_file "/etc/yum.repos.d/docker.repo" do
    source node['docker']['package']['repo_url']
  end
  
  yum_package node['docker']['package']['name']
  
  service "docker.service" do
    provider Chef::Provider::Service::Systemd
    supports :status => true, :restart => true, :reload => true
    action [ :enable, :start ]
  end

when "gentoo"


end
  
