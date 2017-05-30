#!/bin/bash
sudo apt-get -y --force-yes install python g++ fpc
sudo apt-get -y --force-yes install python-dev python-pip python-mysqldb
sudo apt-get -y --force-yes install apache2 php5 mysql-server php5-mysql
sudo pip install pymysql

echo "Creating config files and data ..."
sudo mkdir /home/doj
sudo python setConf.py

sudo cp -R home/* /home/doj
sudo chmod -R 777 /home/doj

echo "Installing doj core ..."

cd lorun
sudo python setup.py install
cd ..

sudo ln -s /home/doj/doj /usr/bin/doj
sudo chmod +x /home/doj/doj

sudo ln -s /home/doj/dojudge /usr/bin/dojudge
sudo chmod +x /home/doj/dojudge

echo "Installing Web UI ..."
sudo mkdir -p /var/www/backup
sudo cp -R /var/www/html/* /var/www/backup
sudo rm -R /var/www/html/*
sudo cp -R web/* /var/www/html/
sudo chmod -R 777 /var/www

echo "Finished. Old web files have been saved in /var/www/backup"
echo "Use: 'sudo doj start' to start judger"
