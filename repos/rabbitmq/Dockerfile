FROM ubuntu
MAINTAINER Justin Plock <jplock@gmail.com>

RUN echo "deb http://archive.ubuntu.com/ubuntu precise main universe" > /etc/apt/sources.list
RUN apt-get update
RUN apt-get upgrade -y

RUN apt-get -y -q install wget logrotate

# Hack for initctl not being available in Ubuntu
RUN dpkg-divert --local --rename --add /sbin/initctl
RUN ln -s /bin/true /sbin/initctl

RUN wget http://www.rabbitmq.com/rabbitmq-signing-key-public.asc
RUN apt-key add rabbitmq-signing-key-public.asc
RUN echo "deb http://www.rabbitmq.com/debian/ testing main" > /etc/apt/sources.list.d/rabbitmq.list
RUN apt-get update
RUN apt-get -y -q install rabbitmq-server || true
RUN /usr/sbin/rabbitmq-plugins enable rabbitmq_management

EXPOSE 5672 15672

CMD ["/usr/sbin/rabbitmq-server"]

