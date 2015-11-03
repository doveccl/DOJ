#!/bin/bash
sudo apt-get -y --force-yes install python g++ fpc
sudo apt-get -y --force-yes install python-dev python-mysqldb
sudo apt-get -y --force-yes install git

echo "Installing doj core ..."

git clone https://github.com/lodevil/Lo-runner.git
cd Lo-runner
sudo python setup.py install
sudo rm -R build
cd ..
sudo rm -R Lo-runner

sudo mkdir /home/doj
sudo python setConf.py

sudo cp core/doj /etc/init.d/
sudo chmod +x /etc/init.d/doj
sudo echo "\n\n\n#DOJ Service Start\nsudo service doj start\n\n\n" >> "/etc/rc.local"

sudo cp core/dojudge /usr/bin
sudo chmod +x /usr/bin/dojudge

echo "Creating config files and data ..."
sudo cp -R home/* /home/doj
sudo chmod -R 777 /home/doj

echo "Installing Web UI ..."
sudo mkdir -p /var/www/backup
sudo cp -R /var/www/html/* /var/www/backup
sudo rm -R /var/www/html/*
sudo cp -R web/* /var/www/html/
sudo chmod -R 777 /var/www

echo "Finished. Old web files have been saved in /var/www/backup"
echo "Use: 'sudo service doj start' to start judger"
