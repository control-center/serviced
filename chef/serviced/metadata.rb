name              "serviced"
maintainer        "Zenoss, Inc."
maintainer_email  "acorley@zenoss.com"
license           ""
description       "Installs serviced dependencies"
long_description  IO.read(File.join(File.dirname(__FILE__), 'README.md'))
version           "0.0.1"

recipe "default", "Installs and configures serviced"

%w{ ubuntu debian fedora centos redhat amazon }.each do |os|
  supports os
end

depends "apt"
depends "yum"
