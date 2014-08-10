Upload
======

Adminstrators can upload this buildpack for everyone to automatically use.

```
zip -r ../staticfile-buildpack.zip *
cf create-buildpack staticfiles_buildpack ../staticfile-buildpack.zip 1
```

Subsequently, update the buildpack with:

```
zip -r ../staticfile-buildpack.zip *
cf update-buildpack staticfiles_buildpack -p ../staticfile-buildpack.zip -i 1
```

Test that it correctly detects the buildpack:

```
cf push staticfile -p test/fixtures/staticfile_app
...
Staging failed: An application could not be detected by any available buildpack
```

Test that it correctly ignores the buildpack if `Staticfile` file is missing:

```
cf push non_staticfile_app -p test/fixtures/non_staticfile_app
```

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
