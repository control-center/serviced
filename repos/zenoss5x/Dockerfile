FROM mattdm/fedora
RUN yum clean all
RUN yum --exclude=systemd\* update -y
RUN yum install -y wget which 
RUN echo "10.175.211.90  artifacts.zenoss.loc" >> /etc/hosts
RUN wget http://artifacts.zenoss.loc/europa/latest/zenoss_resmgr-5.0.0-31.fc18.x86_64.rpm 
RUN yum clean all
RUN yum -y localinstall zenoss_*.rpm
RUN yum -y install net-tools
ADD zenoss_init_pre /opt/zenoss/bin/zenoss_init_pre
ADD global.conf /opt/zenoss/etc/global.conf
ADD install-functions.sh /opt/zenoss/bin/install-functions.sh
RUN chown zenoss:zenoss /opt/zenoss/bin/{zenoss_init_pre,install-functions.sh} /opt/zenoss/etc/global.conf
RUN rm -Rf /opt/zenoss/var/zenpack_actions.txt
RUN sed -i '/GRANT SELECT ON mysql.proc/d' /opt/zenoss/bin/zeneventserver-create-db 
RUN sed -i '/GRANT SELECT ON mysql.proc/d' /opt/zenoss/Products/ZenModel/migrate/fixMysqlPermissions.py 
