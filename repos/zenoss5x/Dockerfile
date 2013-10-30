FROM mattdm/fedora:f19
RUN yum clean all
RUN yum --exclude=systemd\* update -y
RUN rpmdb --rebuilddb
RUN yum install -y wget which 
RUN wget http://artifacts.zenoss.loc/europa/latest/$(wget -O- http://artifacts.zenoss.loc/europa/latest/RPMNAME)
RUN yum clean all
RUN yum -y localinstall zenoss-*.rpm || rpmdb --rebuilddb && yum -y localinstall zenoss-*.rpm
RUN yum -y install net-tools || rpmdb --rebuilddb && yum -y install net-tools
ADD zenoss_init_pre /opt/zenoss/bin/zenoss_init_pre
ADD global.conf /opt/zenoss/etc/global.conf
ADD install-functions.sh /opt/zenoss/bin/install-functions.sh
RUN chown zenoss:zenoss /opt/zenoss/bin/{zenoss_init_pre,install-functions.sh} /opt/zenoss/etc/global.conf
RUN rm -Rf /opt/zenoss/var/zenpack_actions.txt
RUN sed -i '/GRANT SELECT ON mysql.proc/d' /opt/zenoss/bin/zeneventserver-create-db 
RUN sed -i '/GRANT SELECT ON mysql.proc/d' /opt/zenoss/Products/ZenModel/migrate/fixMysqlPermissions.py
ADD zenoss_init.sh /tmp/zenoss_init.sh
RUN /tmp/zenoss_init.sh

