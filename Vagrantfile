# -*- mode: ruby -*-
# vi: set ft=ruby :

# Vagrantfile API/syntax version. Don't touch unless you know what you're doing!
VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|

  config.vm.define "lucid" do |lucid|
    lucid.vm.box = "puppet-lucid"
    lucid.vm.box_url = "http://puppet-vagrant-boxes.puppetlabs.com/ubuntu-server-10044-x64-vbox4210.box"
    lucid.vm.provision "shell", inline: "/vagrant/bin/build_nginx"
  end

  config.vm.define "trusty" do |trusty|
    trusty.vm.box = "ubuntu/trusty64"
    trusty.vm.provision "shell", inline: "/vagrant/bin/build_nginx"
  end
end
