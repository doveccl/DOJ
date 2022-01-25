import * as fs from 'fs'
import * as webpack from 'webpack'
import * as CopyWebpackPlugin from 'copy-webpack-plugin'

export default {
	target: 'node',
	entry: './server',
	mode: 'production',
	output: {
    filename: 'doj.js'
  },
	module: {
		rules: [{
			test: /\.tsx?$/,
			loader: 'ts-loader'
		}]
	},
	resolve: {
		extensions: ['.ts', '.js']
	},
	externals: [(ctx, callback) => {
		if (ctx.request.startsWith('.')) return callback()
		if (!ctx.context.includes('node_modules')) return callback()
		const module = `node_modules/${ctx.request.split('/')[0]}`
		if (fs.existsSync(`${__dirname}/${module}`)) return callback()
		// treat all modules not installed as built-in modules
		callback(null, `commonjs ${ctx.request}`)
	}],
	plugins: [
		new webpack.DefinePlugin({ 'global.GENTLY': false }),
		new CopyWebpackPlugin({
			patterns: [
				{ from: 'testlib', to: 'testlib' },
				{ from: 'mirrorfs.cfg' }
			]
		})
	]
} as webpack.Configuration
