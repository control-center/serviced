FROM mattdm/fedora
RUN yum clean all
RUN yum --exclude=systemd\* update -y
RUN yum install -y wget which 
RUN echo "10.175.211.90  artifacts.zenoss.loc" >> /etc/hosts
RUN wget http://artifacts.zenoss.loc/europa/latest/zenoss_resmgr-5.0.0-31.fc18.x86_64.rpm 
RUN yum clean all
RUN yum -y localinstall zenoss_*.rpm

ADD zenoss_init_pre /opt/zenoss/bin/zenoss_init_pre
