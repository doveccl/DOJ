const { MongoClient } = require('mongodb')

exports.init = async con => {
	const client = await MongoClient.connect(con.url)
	const db = client.db(con.name)
	return col => db.collection(col)
}
