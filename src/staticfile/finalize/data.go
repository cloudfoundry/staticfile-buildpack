package finalize

const (
	initScript = `
# ------------------------------------------------------------------------------------------------
# Copyright 2013 Jordon Bedwell.
# Apache License.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
# except  in compliance with the License. You may obtain a copy of the License at:
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software distributed under the
# License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
# either express or implied. See the License for the specific language governing permissions
# and  limitations under the License.
# ------------------------------------------------------------------------------------------------

export APP_ROOT=$HOME
export LD_LIBRARY_PATH=$APP_ROOT/nginx/lib:$LD_LIBRARY_PATH

mv $APP_ROOT/nginx/conf/nginx.conf $APP_ROOT/nginx/conf/nginx.conf.erb
erb $APP_ROOT/nginx/conf/nginx.conf.erb > $APP_ROOT/nginx/conf/nginx.conf

if [[ ! -f $APP_ROOT/nginx/logs/access.log ]]; then
    mkfifo $APP_ROOT/nginx/logs/access.log
fi

if [[ ! -f $APP_ROOT/nginx/logs/error.log ]]; then
    mkfifo $APP_ROOT/nginx/logs/error.log
fi
`

	startLoggingScript = `
cat < $APP_ROOT/nginx/logs/access.log &
(>&2 cat) < $APP_ROOT/nginx/logs/error.log &
`

	startCommand = `#!/bin/sh
set -ex
$APP_ROOT/start_logging.sh
nginx -p $APP_ROOT/nginx -c $APP_ROOT/nginx/conf/nginx.conf
`

	nginxConfTemplate = `
worker_processes 1;
daemon off;

error_log <%= ENV["APP_ROOT"] %>/nginx/logs/error.log;
events { worker_connections 1024; }

http {
  charset utf-8;
  log_format cloudfoundry '$http_x_forwarded_for - $http_referer - [$time_local] "$request" $status $body_bytes_sent';
  access_log <%= ENV["APP_ROOT"] %>/nginx/logs/access.log cloudfoundry;
  default_type application/octet-stream;
  include mime.types;
  sendfile on;

  gzip on;
  gzip_disable "msie6";
  gzip_comp_level 6;
  gzip_min_length 1100;
  gzip_buffers 16 8k;
  gzip_proxied any;
  gunzip on;
  gzip_static always;
  gzip_types text/plain text/css text/js text/xml text/javascript application/javascript application/x-javascript application/json application/xml application/xml+rss;
  gzip_vary on;

  tcp_nopush on;
  keepalive_timeout 30;
  port_in_redirect off; # Ensure that redirects don't include the internal container PORT - <%= ENV["PORT"] %>
  server_tokens off;

  server {
    listen <%= ENV["PORT"] %>;
    server_name localhost;

    location / {
      root <%= ENV["APP_ROOT"] %>/public;

      {{if .PushState}}
        if (!-e $request_filename) {
          rewrite ^(.*)$ / break;
        }
      {{end}}

      index index.html index.htm Default.htm;

      {{if .DirectoryIndex}}
        autoindex on;
      {{end}}

      {{if .BasicAuth}}
        auth_basic "Restricted";  #For Basic Auth
        auth_basic_user_file <%= ENV["APP_ROOT"] %>/nginx/conf/.htpasswd;
      {{end}}

      {{if .ForceHTTPS}}
        if ($http_x_forwarded_proto != "https") {
          return 301 https://$host$request_uri;
        }
      {{else}}
      <% if ENV["FORCE_HTTPS"] %>
        if ($http_x_forwarded_proto != "https") {
          return 301 https://$host$request_uri;
        }
      <% end %>
      {{end}}

      {{if .SSI}}
        ssi on;
      {{end}}

      {{if .HSTS}}
        add_header Strict-Transport-Security "max-age=31536000";
      {{end}}

      {{if ne .LocationInclude ""}}
        include {{.LocationInclude}};
      {{end}}
    }

  {{if not .HostDotFiles}}
    location ~ /\. {
      deny all;
      return 404;
    }
  {{end}}
  }
}
`
	MimeTypes = `
types {
  text/html html htm shtml;
  text/css css;
  text/xml xml;
  image/gif gif;
  image/jpeg jpeg jpg;
  application/x-javascript js;
  application/atom+xml atom;
  application/rss+xml rss;
  font/ttf ttf;
  font/woff woff;
  font/woff2 woff2;
  text/mathml mml;
  text/plain txt;
  text/vnd.sun.j2me.app-descriptor jad;
  text/vnd.wap.wml wml;
  text/x-component htc;
  text/cache-manifest manifest;
  image/png png;
  image/tiff tif tiff;
  image/vnd.wap.wbmp wbmp;
  image/x-icon ico;
  image/x-jng jng;
  image/x-ms-bmp bmp;
  image/svg+xml svg svgz;
  image/webp webp;
  application/java-archive jar war ear;
  application/mac-binhex40 hqx;
  application/msword doc;
  application/pdf pdf;
  application/postscript ps eps ai;
  application/rtf rtf;
  application/vnd.ms-excel xls;
  application/vnd.ms-powerpoint ppt;
  application/vnd.wap.wmlc wmlc;
  application/vnd.google-earth.kml+xml  kml;
  application/vnd.google-earth.kmz kmz;
  application/x-7z-compressed 7z;
  application/x-cocoa cco;
  application/x-java-archive-diff jardiff;
  application/x-java-jnlp-file jnlp;
  application/x-makeself run;
  application/x-perl pl pm;
  application/x-pilot prc pdb;
  application/x-rar-compressed rar;
  application/x-redhat-package-manager  rpm;
  application/x-sea sea;
  application/x-shockwave-flash swf;
  application/x-stuffit sit;
  application/x-tcl tcl tk;
  application/x-x509-ca-cert der pem crt;
  application/x-xpinstall xpi;
  application/xhtml+xml xhtml;
  application/zip zip;
  application/octet-stream bin exe dll;
  application/octet-stream deb;
  application/octet-stream dmg;
  application/octet-stream eot;
  application/octet-stream iso img;
  application/octet-stream msi msp msm;
  application/json json;
  audio/midi mid midi kar;
  audio/mpeg mp3;
  audio/ogg ogg;
  audio/x-m4a m4a;
  audio/x-realaudio ra;
  video/3gpp 3gpp 3gp;
  video/mp4 mp4;
  video/mpeg mpeg mpg;
  video/quicktime mov;
  video/webm webm;
  video/x-flv flv;
  video/x-m4v m4v;
  video/x-mng mng;
  video/x-ms-asf asx asf;
  video/x-ms-wmv wmv;
  video/x-msvideo avi;
}
`
)
