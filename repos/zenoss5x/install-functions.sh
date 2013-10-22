##############################################################################
# 
# Copyright (C) Zenoss, Inc. 2006, all rights reserved.
# 
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
# 
##############################################################################


#
# Contained below are functions that are associated with installing or
# configuring Zenoss.  These include things like initializing the
# MySQL database or populating the Zeo database.
#

test -r shared-functions.sh && . shared-functions.sh


##
# A zenoss instance is a master configuration when
#  - $ZENHOME/.master exists, or
#  - $ZENHOME/etc/DAEMONS_TXT_ONLY does not exist, or
#  - $ZENHOME/etc/DAEMONS_TXT_ONLY exists and zenjobs is in $ZENHOME/etc/daemons.txt
# 
function is_master()
{
  local daemons_txt_only=$ZENHOME/etc/DAEMONS_TXT_ONLY
  local has_zenjobs=`cat $ZENHOME/etc/daemons.txt 2>/dev/null | sort -u | grep -c "\<zenjobs\>"`

  if [ -e $ZENHOME/.master ] ; then return 0; fi
  if [ ! -e $daemons_txt_only ] ; then return 0; fi
  if [ -e $daemons_txt_only -a "$has_zenjobs" -eq "1" ] ; then return 0; fi
  return 1
}

# Test if amqp global configuration points to localhost
function is_amqp_local()
{
    local amqphost=`$ZENHOME/bin/zenglobalconf -p amqphost`

    if [ "$amqphost" = "localhost" ] ; then return 0; fi
    if [ "$amqphost" = "localhost.localdomain" ] ; then return 0; fi
    if [ "$amqphost" = "127.0.0.1" ] ; then return 0; fi

    return 1
}

# Test for amqp connectivity based on global configuration options
function confirm_amqp_connectivity()
{
    $ZENHOME/bin/python $ZENHOME/Products/ZenUtils/qverify.py --disable-settings-check
}

# confirm flavor of jre
confirm_jre()
{ 
    return
}

# Do nothing for RRD
confirm_rrd()
{ 
    return
}

# return a list of pids and commands of zenoss related processes
get_running()
{
    FORMAT="pid,command"
    if [ "`uname`" = "SunOS" ]; then
        FORMAT="pid,comm"
    fi

    if [ -z "${OS_USERNAME}" ] ; then
        export OS_USERNAME=zenoss
    fi
    id ${OS_USERNAME} >/dev/null 2>&1 || OS_USERNAME=${USER:-${LOGNAME}}

    # NB: buildbot is used to create packages at Zenoss
    ps -o ${FORMAT} -U ${OS_USERNAME} | egrep '[j]ava|[p]yraw|[p]ython|[r]rdcached' | egrep -v 'buildbot|install.sh' | egrep 'zeo|zope|zen'
}

# use ps to see if zenoss processes are running.  fail if any are found
check_running()
{
    PS=`get_running`
    if [ "$PS" != "" ] ; then
        echo
        echo Looks like there are some stray Zenoss processes running.
        echo ----------------------------------------------------------
        get_running
        echo ----------------------------------------------------------
        fail "Please kill these processes before running the install"
    fi
}

# kills off processes related to zenoss
kill_running()
{
    for pid in \
        `get_running | awk '{print $1}'`
    do
        # squelch stderr in case a previous kill terminated current pid
        kill $pid 2>/dev/null
    done
    return 0
}

# create the zodb database
create_zodb_db()
{
    if [ ! -f backup.tar ] ; then
        FORCE=--force
    else
        FORCE=""
    fi

    $ZENHOME/bin/zeneventserver-create-db --dbhost "$ZODB_HOST" --dbport "$ZODB_PORT" --dbname "$ZODB_DB" \
        --dbadminuser "$ZODB_ADMIN_USER" --dbadminpass "$ZODB_ADMIN_PASSWORD" \
        --dbtype "$ZODB_DB_TYPE" $FORCE \
        --dbuser "$ZODB_USER" --dbpass "$ZODB_PASSWORD" --schemadir "$ZENHOME/Products/ZenUtils/relstorage" \
        || fail "Failed to create ZODB database"

}

create_zodb_session_db()
{
    $ZENHOME/bin/zeneventserver-create-db --dbhost "$ZODB_HOST" --dbport "$ZODB_PORT" --dbname "$ZODB_DB"_session \
        --dbadminuser "$ZODB_ADMIN_USER" --dbadminpass "$ZODB_ADMIN_PASSWORD" \
        --dbtype "$ZODB_DB_TYPE" --force \
        --dbuser "$ZODB_USER" --dbpass "$ZODB_PASSWORD" --schemadir "$ZENHOME/Products/ZenUtils/relstorage" \
        || fail "Failed to create ZODB session database"
}

create_zep_db()
{
    $ZENHOME/bin/zeneventserver-create-db --dbhost "$ZEP_HOST" --dbport "$ZEP_PORT" --dbname "$ZEP_DB" \
        --dbadminuser "$ZEP_ADMIN_USER" --dbadminpass "$ZEP_ADMIN_PASSWORD" \
        --dbtype "$ZEP_DB_TYPE" \
        --dbuser "$ZEP_USER" --dbpass "$ZEP_PASSWORD" || fail "Failed to create ZEP database"
}

##
## zeneventserver.conf used to contain the credentials used to access the
## zenoss_zep database, but they have been moved to global.conf. This function
## removes the credentials stored in the old location.
##
remove_zep_jdbc_config()
{
    $PYTHON $ZENHOME/bin/zeneventserver-config -r \
        zep.jdbc.protocol \
        zep.jdbc.hostname \
        zep.jdbc.port \
        zep.jdbc.dbname \
        zep.jdbc.username \
        zep.jdbc.password || fail "Failed to update zeneventserver.conf"
}

configure_amqp() {
    RABBITMQ_ADMIN="`which rabbitmqadmin`"
    if [ ! -z "$RABBITMQ_ADMIN" ]; then
        local user_exists=`"$RABBITMQ_ADMIN" list users | tail -n +4 | egrep -v "^\+" | awk '{ print $2 }' | grep '^'"$RABBITMQ_USER"'$'`
        if [ -z "$user_exists" ]; then
            echo "Adding RabbitMQ user: $RABBITMQ_USER"
            "$RABBITMQ_ADMIN" declare user name="$RABBITMQ_USER" password="$RABBITMQ_PASS" tags=
        fi
        local vhost_exists=`"$RABBITMQ_ADMIN" list vhosts | tail -n +4 | egrep -v "^\+" | awk '{ print $2 }' | grep '^'"$RABBITMQ_VHOST"'$'`
        if [ -z "$vhost_exists" ]; then
            echo "Adding RabbitMQ vhost: $RABBITMQ_VHOST"
            "$RABBITMQ_ADMIN" declare vhost name="$RABBITMQ_VHOST"
        fi
        local perm_exists=`"$RABBITMQ_ADMIN" list permissions user | tail -n +4 | egrep -v "^\+" | awk '{ print $2 }' | grep '^'"$RABBITMQ_USER"'$'`
        if [ -z "$perm_exists" ]; then
            echo "Setting RabbitMQ permissions for user: $RABBITMQ_USER"
            "$RABBITMQ_ADMIN" declare permission vhost="$RABBITMQ_VHOST" user="$RABBITMQ_USER" configure='.*' write='.*' read='.*' 
        fi
    else
        echo "Unable to find rabbitmqadmin. Please refer to the installation"
        echo "guide for instructions on configuring RabbitMQ."
    fi
}

# create a zope instance
run_mkzopeinstance()
{
    echo "make zope instance."
    USERNAME=${OS_USERNAME}
    # sudo version <= 1.6 does not support -n option
    if sudo -n ls 1>/dev/null 2>&1 ; then
        no_pw_prompt_opt="-n"
    fi
    id ${USERNAME} 2>&1 >/dev/null || USERNAME=$USER
    if [ ! -f backup.tar ] ; then
        if [ "$(id -u)" != "0" ]; then
            $ZENHOME/bin/zenglobalconf -s
        else
            su ${USERNAME} -l -c "$ZENHOME/bin/zenglobalconf -s"
        fi
        $PYTHON $ZENHOME/zopehome/mkzopeinstance \
            --dir="$ZENHOME" \
            --user="admin:$ZOPEPASSWORD" || fail Unable to create Zope instance.
        if [ -f $ZENHOME/etc/zenoss.conf ] ; then
            mv $ZENHOME/etc/zenoss.conf $ZENHOME/etc/zope.conf
        fi
        chown root:$USERNAME $ZENHOME/bin/{pyraw,zensocket} 2>/dev/null
        if [ $? -ne 0 -a -n "${no_pw_prompt_opt}" ]; then
        sudo ${no_pw_prompt_opt} chown root:$USERNAME $ZENHOME/bin/{pyraw,zensocket} 2>/dev/null
        if [ $? -ne 0 ]; then
        echo "Make sure to execute:"
                echo "chown root:$USERNAME ${ZENHOME}/bin/{pyraw,zensocket}"
        fi
        fi
        chmod 04750 $ZENHOME/bin/{pyraw,zensocket} 2>/dev/null
        if [ $? -ne 0 -a -n "${no_pw_prompt_opt}" ]; then
            sudo ${no_pw_prompt_opt} chmod 04750 $ZENHOME/bin/{pyraw,zensocket} 2>/dev/null
            if [ $? -ne 0 ]; then
                echo "Make sure to execute: "
                echo "chmod 04750 ${ZENHOME}/bin/{pyraw,zensocket}"
            fi
        fi


        replace "<<INSTANCE_HOME>>" "$ZENHOME" $ZENHOME/etc/zope.conf
        replace "effective-user zenoss" "effective-user $USERNAME" $ZENHOME/etc/zope.conf
        replace "address 8080" "address 9080" $ZENHOME/etc/zope.conf

    fi
}


# load objects into the zodb
run_zenbuild()
{
    if [ ! -f backup.tar ] ; then
        echo Loading initial Zenoss objects into the Zeo database
        echo   '(this can take a few minutes)'
        if [ "$(id -u)" != "0" ]; then
            $ZENHOME/bin/zenbuild $ZENBUILD_ARGS >> $ZENHOME/log/zenbuild.log 2>&1 || fail "Unable to create the initial Zenoss object database"
        else
            su $OS_USERNAME -l -c "$ZENHOME/bin/zenbuild $ZENBUILD_ARGS" >> $ZENHOME/log/zenbuild.log 2>&1 || fail "Unable to create the initial Zenoss object database"
        fi
    else
        run_zenmigrate  >> $ZENHOME/log/zenbuild.log 2>&1 || fail "Data migration failed"
    fi

}

start_zep() {
    echo "Starting zeneventserver..."
    timeout=60
    OS_USERNAME=${OS_USERNAME:-"zenoss"}
    su - "$OS_USERNAME" -c '$ZENHOME/bin/zeneventserver start 1>/dev/null' || return 1
}

start_zencatalogservice() {
    timeout=60
    OS_USERNAME=${OS_USERNAME:-"zenoss"}
    CATSERVICE="${ZENHOME}/bin/zencatalogservice"
    if [ -f "$CATSERVICE" ] ; then
        echo "Starting zencatalogservice..."
        su - "$OS_USERNAME" -c "${CATSERVICE} start 1>/dev/null" || return 1
    fi
}

# migrate the Zenoss objects using the migrate steps in ZenPacks and in base product
run_zenmigrate()
{
    start_zep
    start_zencatalogservice
    echo "Migrating data..."
    $ZENHOME/bin/zenmigrate
}

# starts the Zope server
start_zope()
{
    echo 'Starting Zope Server'

    # This works properly with Zope 2.12 now that ZOPEHOME != ZENHOME
    # Before we were getting "has no attribute 'instancehome' because the
    # zope.conf wasn't getting loaded properly.
    $ZENHOME/bin/zopectl start || fail "Unable to start Zope"

    # $ZENHOME/bin/runzope -C $ZENHOME/etc/zope.conf || fail "Unable to start Zope"
}

check_mysql()
{
    if [ "$USE_ZENDS" = "1" ]; then
        STARTUP_SCRIPTS="/etc/init.d/zends"
    else
        STARTUP_SCRIPTS="/etc/init.d/mysql /etc/init.d/mysqld"
    fi
    for script in $STARTUP_SCRIPTS
    do
        if [ -x ${script} ]; then
            if ! ${script} status 2>/dev/null 1>&2 ; then
                ${script} start
            fi
            return
        fi
    done

    # If we get here we probably don't have MySQL -- doh!
    if [ "$USE_ZENDS" = "1" ];then
        fail "No startup script for ZenDS found -- exiting!"
    else
        fail "No startup script for MySQL found -- exiting!"
    fi
}


# adds localhost as a device via the web UI
add_localhost_device()
{
    ZOPE_HOST=localhost
    if [ -x /usr/bin/host ]
    then
    for h in localhost localhost.localdomain "`hostname`"
    do
        if /usr/bin/host "$h" >/dev/null
        then
                ZOPE_HOST="$h"
            fi
    done
    fi
    HOST_CLASS_PATH="/Server/Linux"
    URL="http://${ZOPE_HOST}:${ZOPE_LISTEN_PORT}/zport/dmd/DeviceLoader/loadDevice?deviceName=${ZOPE_HOST}&devicePath=${HOST_CLASS_PATH}"

    # add localhost as a device
    # wget>=1.11 does basic auth differently, so must use auth-no-challenge
    # option
    if [ -z "`wget --help | grep auth-no-challenge`" ]
    then
        WGET="wget"
    else
        WGET="wget --auth-no-challenge"
    fi
    $WGET -O /dev/null \
        --no-http-keep-alive \
        --user=${ZOPE_USERNAME} \
        --password=${ZOPEPASSWORD} \
        ${URL}

}

# updates the conf files in ${ZENHOME}/etc.  if the user already has a
# conf file then the example conf file will not be laid down on top of
# the existing conf file.  if the conf file already exists it is left
# in place.
update_conf_files()
{
    for EXAMPLE in \
        `ls ${ZENHOME}/etc/*.example`
    do
        CONF=`echo $EXAMPLE | cut -d. -f-2`

        if [ ! -s ${CONF} ]; then
            cp -p $EXAMPLE $CONF
        fi
        chmod 0600 $CONF
    done
}

# In the case of an upgrade, patches runzope and zopectl so they can accept
# alternate CONFIG_FILE parameters. In fresh installs this is already done by
# patching Zope at build time.
patch_zopectl_and_runzope () {
    for f in ${ZENHOME}/bin/zopectl ${ZENHOME}/bin/runzope; do
        sed -i -e 's|^CONFIG_FILE=|[ -z "$CONFIG_FILE" ] \&\& CONFIG_FILE=|' "${f}"
    done
}

# Upgrades a $ZENHOME/etc/zope.conf file to use persistent sessions. If the
# sessions database has been modified in any way from the default, we will not
# try to change it. Otherwise, it will be modified to use the same relstorage
# details as the main database.
upgrade_to_persistent_sessions ()
{
    ZOPECONF=${ZENHOME}/etc/zope.conf

    # Replace the contents of the file with the new stuff
    RESULT=$(${ZENHOME}/bin/python <<EOF
import re, sys

print "Upgrading to use persistent sessions..."

# Get current file contents
with open("${ZOPECONF}", 'r') as f:
    currentfilecontents = f.read()

# Find the session storage
sessionstanza = re.search(r'(?ms)^\s*<zodb_db temporary>.*?</zodb_db>', currentfilecontents).group()

# Verify upgrade needed
if '<temporarystorage>' not in sessionstanza:
    print ("Already using persistent sessions or using custom session storage."
           " $(basename ${ZOPECONF}) will not be altered.")
    sys.exit(0)

# Get database creds from the main database stanza
zodb_db_main = re.search(
    r'(?ms)^\s*<zodb_db main>.*?(%import relstorage.*?</relstorage>).*?</zodb_db>',
    currentfilecontents)

if not zodb_db_main:
    print ("WARNING: Unable to find zodb database credentials in zope.conf. "
           "Please upgrade $(basename ${ZOPECONF}) to use persistent sessions manually.")
    sys.exit(1)

relstoragestanza = zodb_db_main.groups()[0]

# Either modify the actual credentials that are there, or alter the %include
if '<mysql>' in relstoragestanza:
    # 4.x -> 4.1.1
    # Change the database name to add '_session'
    modified_relstorage = re.sub(r'(\s*?db \S*)', r'\1_session', relstoragestanza)
elif '%include zodb_db_main.conf' in relstoragestanza:
    # 4.2 -> 4.2 (developers, mostly)
    # Change the %include to point to the session conf
    modified_relstorage = relstoragestanza.replace('zodb_db_main.conf', 'zodb_db_session.conf')

new_sessionstanza = re.sub(r'(?ms)<temporarystorage>.*</temporarystorage>',
                               modified_relstorage, sessionstanza)
# the session database connection does not work with cache servers
new_sessionstanza = re.sub(r'.*cache-.*', "", new_sessionstanza)

newfilecontents = currentfilecontents.replace(sessionstanza, new_sessionstanza)

with open("${ZOPECONF}", 'w') as f:
    f.write(newfilecontents)

print "${ZOPECONF} has been upgraded to use persistent sessions."

EOF
)

EXITCODE=$?
echo -e "${RESULT}"
return ${EXITCODE}
}


getZeneventServerSetting()
{
   VALUE=`$ZENHOME/bin/python $ZENHOME/bin/zeneventserver-config -p $1`
   if [ ! -n "$VALUE" ]; then
       VALUE=$2
   fi
   echo $VALUE
}

setIfMissing()
{
    KEY=$1
    DEFAULT=$2
    VALUE=`$ZENHOME/bin/python $ZENHOME/bin/zenglobalconf -p $KEY`
    if [ ! -n "$VALUE" ]; then
        echo "adding default ($DEFAULT) for '$KEY' in $ZENHOME/etc/global.conf"
        $ZENHOME/bin/python $ZENHOME/bin/zenglobalconf -u $KEY=$DEFAULT
    fi
}

#
# Migrates settings from pre-4.2 versions of configuration files.
#
# (1) Changes global.conf settings to use new zodb-* prefixes.
# (2) Moves configuration stored in zeneventserver.conf to live in global.conf.
#
upgrade_conf_options_42()
{
    # Upgrades configuration files to use new option names
    pushd $ZENHOME/etc > /dev/null
    for conf in global.conf zen*.conf; do
        if [ -f "$conf" ]; then
            sed -e 's/^dataroot\([ ,    ]*\)/zodb-dataroot\1/g;' \
                -e 's/^cachesize\([ ,   ]*\)/zodb-cachesize\1/g;' \
                -e 's/^host\([ ,        ]*\)/zodb-host\1/g;' \
                -e 's/^port\([ ,        ]*\)/zodb-port\1/g;' \
                -e 's/^mysqluser\([ ,   ]*\)/zodb-user\1/g;' \
                -e 's/^mysqlpasswd\([ , ]*\)/zodb-password\1/g;' \
                -e 's/^mysqldb\([ ,     ]*\)/zodb-db\1/g;' \
                -e 's/^mysqlsocket\([ , ]*\)/zodb-socket\1/g;' \
                -e 's/^cacheservers\([ ,        ]*\)/zodb-cacheservers\1/g;' \
                -e 's/^poll-interval\([ ,       ]*\)/zodb-poll-interval\1/g;' \
                -e 's/^zep_uri\([ ,     ]*\)/zep-uri\1/g;' \
                -i $conf
        fi
    done
    setIfMissing zodb-db-type mysql
    setIfMissing zodb-admin-user root
    setIfMissing zodb-admin-password
    setIfMissing zep-db-type mysql
    setIfMissing zep-host `getZeneventServerSetting zep.jdbc.hostname localhost`
    setIfMissing zep-port `getZeneventServerSetting zep.jdbc.port 3306`
    setIfMissing zep-db `getZeneventServerSetting zep.jdbc.dbname zenoss_zep`
    setIfMissing zep-user `getZeneventServerSetting zep.jdbc.zenoss.username zenoss`
    setIfMissing zep-password `getZeneventServerSetting zep.jdbc.password zenoss`
    setIfMissing zep-admin-user root
    setIfMissing zep-admin-password
    popd > /dev/null

    # Migrate options from zeneventserver.conf to global.conf
    for zep_opt in "zep.jdbc.hostname|zep-host" \
                   "zep.jdbc.port|zep-port" \
                   "zep.jdbc.dbname|zep-db" \
                   "zep.jdbc.username|zep-user" \
                   "zep.jdbc.password|zep-password"; do
        local old_name=`echo "$zep_opt" | cut -d'|' -f1`
        local new_name=`echo "$zep_opt" | cut -d'|' -f2`
        current_val=`$ZENHOME/bin/python $ZENHOME/bin/zeneventserver-config -p "$old_name"`
        if [ -n "$current_val" ]; then
            $ZENHOME/bin/python $ZENHOME/bin/zenglobalconf -u "$new_name=$current_val"
            $ZENHOME/bin/python $ZENHOME/bin/zeneventserver-config -r "$old_name"
        fi
    done
}

# Find all the root-owned files & directories under $ZENHOME and change them to something reasonable.
# Typically this is 'zenoss:zenoss'.
#
# The group may be specified as an optional second parameter.  Otherwise it defaults to the
# primary group associated with the specified owner.
#
fix_zenhome_owner_and_group()
{
    owner=$1

    # Really only makes sense to be remediating ownership if we're currently running/installing
    # as root.

    if [ "`id -u`" = "0" ];then
        if [ -z "${ZENHOME}" ];then
            fail "\$ZENHOME not set.  Unable to fix file ownership under \$ZENHOME."
        fi

        if [ "${ZENHOME}" = "/" ];then
            fail "\$ZENHOME unwisely set to /.  Abandoning request to change file ownership under /."
        fi

        if [ -z "${owner}" ];then
            fail "Null owner parameter passed to fix_zenhome_owner().  Unable to fix file ownership under ${ZENHOME}."
        else
            if id ${owner} 2>/dev/null 1>&2 ;then
                # If group is not explicitly passed in, then default to the effective group for the owner.
                group=${2:-`id -gn ${owner}`}

                # All pre-reqs validated.
                # perform recursive chown, 100x faster than find, iterate, chown each file
                chown -Rf ${owner}:${group} ${ZENHOME}/*
                chown root:${group} ${ZENHOME}/bin/{zensocket,pyraw}
                chmod 04750 $ZENHOME/bin/{zensocket,pyraw}
            else
                fail "Invalid owner (${owner}) passed to fix_zenhome_owner().  Unable to fix file ownership under ${ZENHOME}."
            fi
        fi
    fi
}

register_zproxy() 
{
    echo "Validating redis is running"
    msg=`redis-cli FLUSHALL`
    if [ "$?" -ne 0 ]; then
        fail "Failed to connect to redis: ${msg}"
    else
        msg=`su - zenoss -c "zproxy register load-scripts"`
    	if [ "$?" -ne 0 ]; then
        	fail "Failed to load proxy scripts: ${msg}"
        fi
        msg=`su - zenoss -c "zproxy register from-file ${ZENHOME}/etc/zproxy_registration.conf"`
        if [ "$?" -ne 0 ]; then
           	fail "Failed to load proxy registrations: ${msg}"
        fi
    fi
}
