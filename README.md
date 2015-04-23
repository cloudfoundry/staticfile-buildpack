Deploy static HTML/JS/CSS apps to Cloud Foundry
-----------------------------------------------

Working on a pure front-end only web app or demo? It is easy to share it via your Cloud Foundry:

```
cf push my-site -m 64M -b https://github.com/cloudfoundry/staticfile-buildpack.git
```

Your Cloud Foundry might already have this buildpack installed (see [Upload](#administrator-upload) section for administration):

```
$ cf buildpacks
Getting buildpacks...

buildpack          position   enabled   locked   filename
staticfiles        1          true      false    staticfile-buildpack-v0.4.2.zip
java_buildpack     2          true      false    java-buildpack-offline-v2.4.zip
...
```

You only need to create a `Staticfile` file for Cloud Foundry to detect this buildpack:

```
touch Staticfile
cf push my-site -m 64M
```

Why `-m 64M`? Your static assets will be served by [Nginx](http://nginx.com/) and it only requires 20M \[[reference](http://wiki.nginx.org/WhyUseIt)]. The `-m 64M` reduces the RAM allocation from the default 1G allocated to Cloud Foundry containers. In the future there may be a way for a buildpack to indicate its default RAM requirements; but not as of writing.

Configuration
=============

### Alternate root folder

By default, the buildpack will serve `index.html` and all other assets from the root folder of your project.

In many cases, you may have an alternate folder where your HTML/CSS/JavaScript files are to be served from, such as `dist/` or `public/`.

To configure the buildpack add the following line to your `Staticfile`:

```yaml
root: dist
```

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

### Directory Index

If your site doesn't have a nice `index.html`, you can configure `Staticfile` to display a Directory Index of other files; rather than show a relatively unhelpful 404 error.

![index](http://cl.ly/image/2U2y121g000g/directory-index.png)

Add a line to your `Staticfile` that begins with `directory:`

```
directory: visible
```

### Advanced Nginx configuration

You can customise the Nginx configuration further, by adding `nginx.conf` and/or `mime.types` to your root folder.

If the buildpack detects either of these files, they will be used in place of the built-in versions. See the default [nginx.conf](https://github.com/cloudfoundry-incubator/staticfile-buildpack/blob/master/conf/nginx.conf) and [mime.types](https://github.com/cloudfoundry-incubator/staticfile-buildpack/blob/master/conf/nginx.conf) files for inspiration.

Administrator Upload
====================

Everyone can automatically use this buildpack if your Cloud Foundry Administrator uploads it.

[Releases](https://github.com/cloudfoundry/staticfile-buildpack/releases) are publicly downloadable.

To initially install, say v0.5.1:

```
wget https://github.com/cloudfoundry/staticfile-buildpack/releases/download/v0.5.1/staticfile-buildpack-v0.5.2.zip
cf create-buildpack staticfiles_buildpack staticfile-buildpack-v0.5.1.zip 1
```

Subsequently update the buildpack, say v0.9.9:

```
wget https://github.com/cloudfoundry/staticfile-buildpack/releases/download/v0.9.9/staticfile-buildpackv0.9.9.zip
cf update-buildpack staticfiles_buildpack -p staticfile-buildpackv0.9.9.zip
```

### To create/upload from source repository

1. Get latest buildpack dependencies

  ```shell
  BUNDLE_GEMFILE=cf.Gemfile bundle
  ```

1. Build the buildpack

  ```shell
  BUNDLE_GEMFILE=cf.Gemfile bundle exec buildpack-packager [ cached | uncached ]
  ```

1. Use in Cloud Foundry

  Upload the buildpack to your Cloud Foundry and optionally specify it by name
  
  ```bash
  cf create-buildpack custom_node_buildpack node_buildpack-offline-custom.zip 1
  cf push my_app -b custom_node_buildpack
  ```


Building Nginx
==============

```
vagrant up
```

Vagrant will spin up two machines, one lucid and one trusty and call the buildscript located in bin/build_nginx

Nginx will be stuffed into a tarball in the `vendor/` folder.

Finally, destroy the vagrant VM:

```
vagrant destroy
```


Reporting Issues
================

Open an issue on this project.

Active Development
==================

The project backlog is on [Pivotal Tracker](https://www.pivotaltracker.com/projects/1042066).

Acknowledgements
================

This buildpack is based heavily upon Jordon Bedwell's Heroku buildpack and the modifications by David Laing for Cloud Foundry [nginx-buildpack](https://github.com/cloudfoundry-community/nginx-buildpack). It has been tuned for usability (configurable with `Staticfile`) and to be included as a default buildpack (detects `Staticfile` rather than the presence of an `index.html`). Thanks for the buildpack Jordon!
