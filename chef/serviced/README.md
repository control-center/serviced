# serviceD Cookbook

## Basic Usage
* Simply add serviced to your runlist, all defaults are taken care of!

## Advanced Usage

### Vagrant Files
* ../vagrant/precise64/Vagrantfile
* ../vagrant/fedora19/Vagrantfile

### FS Setup
* Make a new directory to hold your Vagrant workspace (/opt/vagrant/serviced)
* Place the appropriate Vagrantfile in the root of the workspace (/opt/vagrant/serviced/Vagrantfile)
* Create a new directory called 'cookbooks' (/opt/vagrant/serviced/cookbooks/)
* Place this serviced directory in the cookbook location (/opt/vagrant/serviced/cookbooks/serviced/)
* Place any dependencies by chdir to /opt/vagrant/serviced/cookbooks and do the following: "knife cookbook site download (name); tar -zxvf (name).tgz;rm (name).tgz" on [apt,build-essential,chef_handler,dmg,docker,dpkg_autostart,git,golang,lxc,modules,runit,windows,yum]
* Run 'vagrant up' in the root of the namespace
