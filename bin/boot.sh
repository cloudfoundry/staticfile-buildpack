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
export LD_LIBRARY_PATH=$APP_ROOT/openresty/lib:$LD_LIBRARY_PATH

conf_file=$APP_ROOT/openresty/nginx/conf/nginx.conf

echo $(env)

erb $conf_file > $APP_ROOT/openresty/nginx/conf/nginx-final.conf

# ------------------------------------------------------------------------------------------------

touch $APP_ROOT/openresty/nginx/logs/access.log
touch $APP_ROOT/openresty/nginx/logs/error.log

(tail -f -n 0 $APP_ROOT/openresty/nginx/logs/*.log &)
exec $APP_ROOT/openresty/nginx/sbin/nginx -p $APP_ROOT/openresty/nginx -c $APP_ROOT/openresty/nginx/conf/nginx-final.conf

# ------------------------------------------------------------------------------------------------
