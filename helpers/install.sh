#!/bin/bash


# tested on fresh ubuntu 20.04

apt install git gcc

wget https://go.dev/dl/go1.17.12.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.17.12.linux-amd64.tar.gz
ln -s /usr/local/go/bin/go /usr/sbin/go
go version


echo UPDATING LOGGING
ln -sf /opt/quepasa-source/helpers/syslog.conf /etc/rsyslog.d/10-quepasa.conf

echo UPDATING LOGROTATE
ln -sf /opt/quepasa-source/helpers/quepasa.logrotate.d /etc/logrotate.d/quepasa

/bin/mkdir -p /var/log/quepasa
/bin/chmod 755 /var/log/quepasa
/bin/chown syslog:adm /var/log/quepasa

echo RESTARTING SERVICES
systemctl restart rsyslog

echo UPDATING SYSTEMD SERVICE
ln -sf /opt/quepasa-source/helpers/quepasa.service /etc/systemd/system/quepasa.service
systemctl daemon-reload

adduser --disabled-password --gecos "" -home /opt/quepasa quepasa
chown -R quepasa /opt/quepasa-source



cp /opt/quepasa-source/helpers/.env /opt/quepasa/.env

systemctl enable quepasa.service
systemctl start quepasa



# Setup Quepasa user
echo "Setup Quepasa user >>>  http://<your-ip>:31000/setup"


exit 0