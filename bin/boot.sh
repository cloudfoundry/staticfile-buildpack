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

export LD_LIBRARY_PATH=$APP_ROOT/openresty/luajit/lib:$LD_LIBRARY_PATH

export PATH=$APP_ROOT/openresty/bin:$PATH

export JWT_SECRET="vkdbfkvjbdfkv"

export BASIC_AUTH_SECRET="fdkvbhdfjhvbjdfbvj"

mkdir -p /tmp/staged/app/openresty/nginx/logs
touch /tmp/staged/app/openresty/nginx/logs/error.log
mkdir -p /tmp/staged/app/openresty/nginx/client_body_temp

cp -a $APP_ROOT/openresty/lualib /tmp/staged/app/openresty/lualib/

mkdir -p $APP_ROOT/nginx/logs
touch $APP_ROOT/nginx/logs/error.log

conf_file=$APP_ROOT/openresty/nginx/conf/nginx.conf
if [ -f $APP_ROOT/public/nginx.conf ]
then
  conf_file=$APP_ROOT/public/nginx.conf
fi

mv $conf_file $APP_ROOT/openresty/nginx/conf/orig.conf
erb $APP_ROOT/openresty/nginx/conf/orig.conf > $APP_ROOT/openresty/nginx/conf/nginx.conf

# ------------------------------------------------------------------------------------------------

# mkfifo $APP_ROOT/openresty/nginx/logs/access.log
# mkfifo $APP_ROOT/openresty/nginx/logs/error.log
#
# cat < $APP_ROOT/openresty/nginx/logs/access.log &
# (>&2 cat) < $APP_ROOT/openresty/nginx/logs/error.log &

exec $APP_ROOT/openresty/nginx/sbin/nginx -c $APP_ROOT/openresty/nginx/conf/nginx.conf

# ------------------------------------------------------------------------------------------------
