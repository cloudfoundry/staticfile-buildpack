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
echo "====> in boot.sh"
export APP_ROOT=$HOME
# export LD_LIBRARY_PATH=$APP_ROOT/nginx/lib:$LD_LIBRARY_PATH

export LD_LIBRARY_PATH=$APP_ROOT/openresty/luajit/lib:$LD_LIBRARY_PATH

# conf_file=$APP_ROOT/openresty/nginx/conf/nginx.conf
# if [ -f $APP_ROOT/public/nginx.conf ]
# then
#   conf_file=$APP_ROOT/public/nginx.conf
# fi
#
# mv $conf_file $APP_ROOT/openresty/nginx/conf/orig.conf
# erb $APP_ROOT/openresty/nginx/conf/orig.conf > $APP_ROOT/openresty/nginx/conf/nginx.conf


# mkdir -p $APP_ROOT/nginx/nginx/client_body_temp
# mkdir -p $APP_ROOT/nginx/logs
# cat > $APP_ROOT/nginx/logs/error.log
# cat > $APP_ROOT/nginx/nginx/error.log
# cat > $APP_ROOT/nginx/nginx/access.log
# ------------------------------------------------------------------------------------------------

# mkfifo $APP_ROOT/openresty/nginx/logs/access.log
# mkfifo $APP_ROOT/openresty/nginx/logs/error.log
#
# cat < $APP_ROOT/openresty/nginx/logs/access.log &
# (>&2 cat) < $APP_ROOT/openresty/nginx/logs/error.log &
OPENRESTY_PREFIX=$APP_ROOT/openresty
NGINX_PREFIX=$APP_ROOT/openresty/nginx


export PATH="$PATH:/usr/local/sbin:/usr/sbin/:/sbin"

status "==> Downloading OpenResty..." \
 && curl -sSL http://openresty.org/download/ngx_openresty-${OPENRESTY_VERSION}.tar.gz | tar -xvz \
 && echo "==> Configuring OpenResty..." \
 && cd ngx_openresty-* \
 && readonly NPROC=$(grep -c ^processor /proc/cpuinfo 2>/dev/null || 1) \
 && echo "using upto $NPROC threads" \
 && ./configure \
    --prefix=$OPENRESTY_PREFIX \
    --http-client-body-temp-path=$NGINX_PREFIX/client_body_temp \
    --http-log-path=$NGINX_PREFIX/logs/access.log \
    --error-log-path=$NGINX_PREFIX/logs/error.log \
    --with-luajit \
    --with-pcre-jit \
    --with-ipv6 \
    --with-http_ssl_module \
    --without-http_ssi_module \
    --without-http_userid_module \
    --without-http_fastcgi_module \
    --without-http_uwsgi_module \
    --without-http_scgi_module \
    --without-http_memcached_module \
    -j${NPROC} \
 && echo "==> Building OpenResty..." \
 && make -j${NPROC} \
 && echo "==> Installing OpenResty..." \
 && make install \
 && echo "==> Finishing..." \


exec $APP_ROOT/openresty/nginx/sbin/nginx -c $APP_ROOT/openresty/nginx/conf/nginx.conf

# ------------------------------------------------------------------------------------------------
