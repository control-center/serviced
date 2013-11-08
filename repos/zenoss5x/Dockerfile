FROM mattdm/fedora:f19
RUN yum clean all
RUN yum --exclude=systemd\* update -y
RUN rpmdb --rebuilddb && yum install -y  which net-tools \
	MySQL-python \
	binutils \
	dmidecode \
	gnupg \
	gzip \
	hiredis \
	libaio \
	libcap \
	libgomp \
	libpcre \
	libsmi \
	libssl \
	libutil \
	libxslt \
	libz \
	mariadb \
	mariadb-server \
	net-snmp \
	net-snmp-libs \
	net-snmp-utils \
	net-tools \
	nmap \
	openssl \
	openssl-devel \
	patch \
	pcre-devel \
	protobuf-python \
	rabbitmq-server \
	readline \
	redis \
	rsync \
	sudo \
	supervisor \
	sysstat \
	tar \
	wget \
	python \
	python-ZSI \
	python-amqplib \
	python-billiard \
	python-celery \
	python-decorator \
	python-eventlet \
	python-httplib2 \
	python-imaging \
	python-ipaddr \
	python-ldap \
	python-lxml \
	python-memcached \
	python-networkx \
	python-redis \
	python-six \
	python-sphinx \
	python-suds \
	python-twisted \
	python-txamqp \
	python-ujson \
	python-urllib3 \
	alsa-lib \
	bind-libs \
	bind-license \
	bind-utils \
	flac-libs \
	gsm \
	java-1.7.0-openjdk \
	javapackages-tools \
	jline \
	jpackage-utils \
	libXtst \
	libart_lgpl \
	libasyncns \
	libgcj \
	libogg \
	libsndfile \
	libtirpc \
	libvorbis \
	nagios-common \
	nagios-plugins \
	nagios-plugins-dig \
	nagios-plugins-dns \
	nagios-plugins-http \
	nagios-plugins-ircd \
	nagios-plugins-ldap \
	nagios-plugins-ntp \
	nagios-plugins-perl \
	nagios-plugins-ping \
	nagios-plugins-rpc \
	nagios-plugins-tcp \
	pulseaudio-libs \
	rhino \
	rpcbind \
	ttmkfdir \
	tzdata-java \
	xorg-x11-fonts-Type1 \
	zip \
	&& yum clean all

ENV RABBITMQ_NODENAME rabbit@localhost
RUN rpmdb --rebuilddb && yum install -y http://artifacts.zenoss.loc/europa/53/zenoss-5.0.0-57.fc19.x86_64.rpm
RUN yum -y localinstall zenoss-*.rpm || rpmdb --rebuilddb && yum -y localinstall zenoss-*.rpm
ADD zenoss_init_pre /opt/zenoss/bin/zenoss_init_pre
ADD global.conf /opt/zenoss/etc/global.conf
ADD install-functions.sh /opt/zenoss/bin/install-functions.sh
RUN chown zenoss:zenoss /opt/zenoss/bin/{zenoss_init_pre,install-functions.sh} /opt/zenoss/etc/global.conf
RUN sed -i '/GRANT SELECT ON mysql.proc/d' /opt/zenoss/bin/zeneventserver-create-db 
RUN sed -i '/GRANT SELECT ON mysql.proc/d' /opt/zenoss/Products/ZenModel/migrate/fixMysqlPermissions.py

ADD zenoss_init.sh /tmp/zenoss_init.sh
RUN /tmp/zenoss_init.sh
