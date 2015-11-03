#!/usr/bin/env python
#coding: utf-8

import MySQLdb, os

db_user = ''
db_pwd = ''

def tryConn():
	global db_user, db_pwd

	try:
		conn = MySQLdb.connect(host='localhost', user=db_user, passwd=db_pwd)
		return True
	except MySQLdb.Error, e:
		print 'Mysql Error %d: %s' % (e.args[0], e.args[1])
		return False

def getConf():
	global db_user, db_pwd

	db_user = raw_input('Enter MySQL Username:')
	db_pwd = raw_input('Enter MySQL Password:')
	if db_user == '':
		db_user = 'root'
	if db_pwd == '':
		db_pwd = 'root'

	return tryConn()

if __name__ == '__main__':
	while not getConf():
		print 'Unable to connect to MySQL'
		print 'Please check your input and enter again'
		print 'You have entered %s %s' % (db_user, db_pwd)

	conf = file('/home/doj/db.conf', 'w')
	conf.write('''{
	"db_host": "localhost",
	"db_user": "%s",
	"db_pwd": "%s",
	"db_name": "doj"
}''' % (db_user, db_pwd))
	conf.close()
	os.system("mysql -u%s -p%s < doj.sql" % (db_user, db_pwd))
