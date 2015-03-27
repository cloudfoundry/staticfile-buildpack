Deploy static HTML/JS/CSS apps to Cloud Foundry
-----------------------------------------------

Working on a pure front-end only web app or demo? It is easy to share it via your Cloud Foundry:

```
cf push my-site -m 64M -b https://github.com/cloudfoundry-community/staticfile-buildpack.git
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

[Releases](https://github.com/cloudfoundry-community/staticfile-buildpack/releases) are publicly downloadable.

To initially install, say v0.4.1:

```
wget https://github.com/cloudfoundry-community/staticfile-buildpack/releases/download/v0.4.2/staticfile-buildpack-v0.4.2.zip
cf create-buildpack staticfiles_buildpack staticfile-buildpack-v0.4.2.zip 1
```

Subsequently update the buildpack, say v0.5.0:

```
wget https://github.com/cloudfoundry-community/staticfile-buildpack/releases/download/v0.5.0/staticfile-buildpack-v0.5.0.zip
cf update-buildpack staticfiles_buildpack -p staticfile-buildpack-v0.5.0.zip
```

### To create/upload from source repository

```
zip -r ../staticfile-buildpack.zip *
cf create-buildpack staticfiles_buildpack ../staticfile-buildpack.zip 1
```

Subsequently, update the buildpack with:

```
zip -r ../staticfile-buildpack.zip *
cf update-buildpack staticfiles_buildpack -p ../staticfile-buildpack.zip
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

Acceptance Tests
----------------

After installing the buildpack, you can run a set of Acceptance Tests.

https://github.com/cloudfoundry-community/staticfile-buildpack-acceptance-tests

Local development
=================

There are five example apps that should all compile successfully:

```
cf create-space staticfile-tests
STACK="lucid64"
BUILDPACK="https://github.com/cloudfoundry-incubator/staticfile-buildpack"
cf push staticfile -p test/fixtures/staticfile_app -b $BUILDPACK -s $STACK --random-route
cf open staticfile

cf push staticfile -p test/fixtures/alternate_root -b $BUILDPACK -s $STACK --random-route
cf open staticfile

cf push staticfile -p test/fixtures/directory_index -b $BUILDPACK -s $STACK --random-route
cf open staticfile

cf push staticfile -p test/fixtures/basic_auth -b $BUILDPACK -s $STACK --random-route
cf open staticfile

cf push staticfile -p test/fixtures/reverse_proxy -b $BUILDPACK -s $STACK --random-route
cf open staticfile

cf delete-space staticfile-tests
```

You can test for other stacks using:

```
STACK=cflinuxfs2
```

You can test someone's pull request branch, say https://github.com/cloudfoundry-incubator/staticfile-buildpack/pull/27, using:

```
BUILDPACK="https://github.com/simonjohansson/staticfile-buildpack#cflinuxfs2"
```

Note: the `#cflinuxfs2` is the name of the branch on github for the pull request.

Building Nginx
==============

```
vagrant up
vagrant ssh
```

Inside vagrant:

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

Buildpack release process
=========================

Each tagged release should include an uploaded `staticfile-buildpack-vX.Y.Z.zip` to Github to make it easy to download by administrators.

These instructions use the [github-release](https://github.com/aktau/github-release) tool.

```
tag=vX.Y.Z
description="USEFUL DESCRIPTION"
git tag $tag
git push --tags
github-release release \
    --user cloudfoundry-community \
    --repo staticfile-buildpack \
    --tag $tag \
    --name "Staticfile Buildpack $tag" \
    --description "$description"

zip -r ../staticfile-buildpack-$tag.zip *

github-release upload \
    --user cloudfoundry-community \
    --repo staticfile-buildpack \
    --tag $tag \
    --name staticfile-buildpack-$tag.zip \
    --file ../staticfile-buildpack-$tag.zip
```

Acknowledgements
================

This buildpack is based heavily upon Jordon Bedwell's Heroku buildpack and the modifications by David Laing for Cloud Foundry [nginx-buildpack](https://github.com/cloudfoundry-community/nginx-buildpack). It has been tuned for usability (configurable with `Staticfile`) and to be included as a default buildpack (detects `Staticfile` rather than the presence of an `index.html`). Thanks for the buildpack Jordon!
