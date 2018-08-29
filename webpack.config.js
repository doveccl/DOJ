const path = require('path')
const config = require('config')
const webpack = require('webpack')
const HtmlWebpackPlugin = require('html-webpack-plugin')

const API_PORT = config.get('port')

module.exports = (env, argv) => {
	const dev = argv.mode === 'development'
	const prod = argv.mode === 'production'

	const config = {
		entry: './web/index.tsx',
		output: {
			path: path.resolve(__dirname, './dist'),
			filename: '[name].js'
		},
		module: {
			rules: [
				{
					test: /\.tsx?$/,
					loader: 'babel-loader!ts-loader'
				},
				{
					test: /\.less$/,
					loader: 'style-loader!css-loader!less-loader'
				}
			]
		},
		resolve: {
			extensions: [ '.tsx', '.ts', '.js' ]
		},
		externals: {
			'react': 'React',
			'react-dom': 'ReactDOM'
		},
		plugins: [
			new HtmlWebpackPlugin({
				filename: 'index.html',
				template: `web/${argv.mode}.html`
			})
		]
	}

	if (dev) {
		config.devtool = 'source-map'
		config.plugins.push(
			new webpack.HotModuleReplacementPlugin(),
			new webpack.NamedModulesPlugin()
		)
		config.devServer = {
			hot: true,
			proxy: {
				'/api': `http://localhost:${API_PORT}`
			}
		}
	}

	return config
}
