Deploy static HTML/JS/CSS apps to Cloud Foundry
-----------------------------------------------

Working on a pure front-end only web app or demo? It is easy to share it via your Cloud Foundry:

```
cf push my-site -m 64M -b https://github.com/drnic/staticfile-buildpack.git
```

With your administrators blessing, the buildpack can be uploaded for everyone to use (see [Upload](#upload) section below). Then you simply need a `Staticfile` file for Cloud Foundry to detect this buildpack:

```
touch Staticfile
cf push my-site -m 64M
```

Your static assets will be served by [Nginx](http://nginx.com/) and it only requires 20M [[reference](http://wiki.nginx.org/WhyUseIt)]; rather than the default 1G allocated to Cloud Foundry containers.

Configuration
=============

### Basic authentication

Protect your website with a user/password configured via environment variables.

![basic-auth](http://cl.ly/image/13402a2d0R1i/basicauth.png)

Convert the username / password to the required format: http://www.htaccesstools.com/htpasswd-generator/

For example, username `bob` and password `bob` becomes `bob:$apr1$DuUQEQp8$ZccZCHQElNSjrg.erwSFC0`.

Create a file in the root of your application `Staticfile.auth`. This becomes the `.htpasswd` file for nginx to project your site. It can include one or more user/password lines.

```
bob:$apr1$DuUQEQp8$ZccZCHQElNSjrg.erwSFC0
```

Push your application to apply changes to basic auth. Remove the file and push to disable basic auth.

Upload
======

Adminstrators can upload this buildpack for everyone to automatically use.

```

zip -r ../staticfile-buildpack.zip * cf create-buildpack staticfiles_buildpack ../staticfile-buildpack.zip 1

```

Subsequently, update the buildpack with:

```

zip -r ../staticfile-buildpack.zip * cf update-buildpack staticfiles_buildpack -p ../staticfile-buildpack.zip -i 1

```

Test that it correctly detects the buildpack:

```

cf push staticfile -p test/fixtures/staticfile_app ... Staging failed: An application could not be detected by any available buildpack

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

vagrant up vagrant ssh

```

```

cd /vagrant ./bin/build_nginx exit

```

Nginx will be stuffed into a tarball in the `vendor/` folder.

Finally, destroy the vagrant VM:

```

vagrant destroy```

Acknowledgements
================

This buildpack is based heavily upon Jordon Bedwell's [nginx-buildpack](https://github.com/cloudfoundry-community/nginx-buildpack). It has been tuned for usability (configurable with `Staticfile`) and to be included as a default buildpack (detects `Staticfile` rather than the presence of an `index.html`). Thanks for the buildpack Jordon!
