const path = require('path')
const config = require('config')
const webpack = require('webpack')
const HtmlWebpackPlugin = require('html-webpack-plugin')
const MiniCssExtractPlugin = require("mini-css-extract-plugin")

const API_PORT = config.get('port')

module.exports = (env, argv) => {
	const dev = argv.mode === 'development'
	const prod = argv.mode === 'production'

	const config = {
		entry: {
			app: './web'
		},
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
					use: [
						MiniCssExtractPlugin.loader,
						'css-loader',
						'less-loader?javascriptEnabled'
					]
				}
			]
		},
		resolve: {
			extensions: ['.tsx', '.ts', '.js']
		},
		plugins: [
			new HtmlWebpackPlugin({
				title: 'DOJ',
				favicon: 'web/logo.png',
				meta: {
					viewport: 'width=device-width, initial-scale=1'
				}
			}),
			new MiniCssExtractPlugin({
				filename: '[name].css'
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
			historyApiFallback: true,
			proxy: {
				'/api': `http://localhost:${API_PORT}`
			}
		}
	}

	return config
}
