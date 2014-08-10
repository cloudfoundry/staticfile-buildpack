Local development
=================

```
cf push staticfile -p test/fixtures/staticfile_app -b https://github.com/drnic/staticfile-buildpack.git
```

Building Nginx
==============

```
vagrant up
vagrant ssh
```

```
cd /vagrant
./bin/build_nginx
exit
```

Nginx will be stuffed into a tarball in the `vendor/` folder.

Finally, destroy the vagrant VM:

```
vagrant destroy
```
