import fs from 'fs'
import webpack from 'webpack'
import CopyWebpackPlugin from 'copy-webpack-plugin'

const config: webpack.Configuration = {
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
		if (ctx.request.startsWith('.'))
			return callback()
		else if (!ctx.context.includes('node_modules'))
			return callback()
		else if (['formidable'].includes(ctx.request))
			return callback(null, `commonjs ${ctx.request}`)
		const module = `node_modules/${ctx.request.split('/')[0]}`
		if (fs.existsSync(`${__dirname}/${module}`)) return callback()
		// treat all modules not installed as built-in modules
		callback(null, `commonjs ${ctx.request}`)
	}],
	plugins: [
		new CopyWebpackPlugin({
			patterns: [
				{ from: 'testlib', to: 'testlib' },
				{ from: 'mirrorfs.cfg' }
			]
		})
	]
}

export default config
