# Change Log

## v0.5.2

### Features

- Respond with `Content-Type: application/json` for `.json` files. #30 [Thanks, @danielsiwiec!]


## v0.5.1

- Include gzip static module in nginx [thanks @ljfranklin]
- Allow alternate root to be `public` folder


## v0.5.0

- Support for `cflinuxfs2` trusty stack (and continued support for `lucid64` stack) [thanks @simonjohansson]
- Remove trailing whitespace from Staticfile 'root:' value [thanks @edmorley]
- Use rsync rather than mv to ensure correct files present in public/ [thanks @edmorley]
- add text/cache-manifest mime type for .manifest files [thanks @hairmare]
- Ensure that trailing slash redirects don't include `ENV[PORT]` [thanks @mrdavidlaing]

### Testing buildpacks

There is now a basic test harness script in `tests/test.sh`.

To test a branch on github:

```
ORG=edmorley BRANCH="root_dir-whitespace" ./test/test.sh
```
