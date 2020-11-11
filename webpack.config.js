const path = require('path')
const WebpackCdnPlugin = require('webpack-cdn-plugin')
const HtmlWebpackPlugin = require('html-webpack-plugin')
const MiniCssExtractPlugin = require("mini-css-extract-plugin")
const OptimizeCssAssetsPlugin = require('optimize-css-assets-webpack-plugin')

const packageJson = require('./package.json')

module.exports = (env, argv) => {
	const dev = argv.mode === 'development'
	const prod = argv.mode === 'production'
	const min = prod ? '.min' : ''
	const reactMode = prod ? 'production.min' : 'development'

	const config = {
		entry: {
			app: './web'
		},
		output: {
			publicPath: '/',
			path: path.resolve(__dirname, './dist'),
			filename: '[name].[hash:8].js',
			chunkFilename: '[name].chunk.[chunkhash:8].js'
		},
		module: {
			rules: [
				{
					test: /\.tsx?$/,
					use: [
						'babel-loader',
						'ts-loader'
					]
				},
				{
					test: /\.(c|le)ss$/,
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
					author: packageJson.author,
					description: packageJson.description,
					keywords: packageJson.keywords.join(',')
				}
			}),
			new WebpackCdnPlugin({
				modules: [
					{ name: 'moment', path: `min/moment.min.js` },
					{ name: 'axios', path: `dist/axios${min}.js` },
					{ name: 'ace-builds', var: 'ace', paths: [`src-min-noconflict/ace.js`, `src-min-noconflict/ext-static_highlight.js`] },
					{ name: 'katex', path: `dist/katex${min}.js`, style: `dist/katex${min}.css` },
					{ name: 'socket.io-client', var: 'io', path: 'dist/socket.io.js' },
					{ name: 'react', var: 'React', path: `umd/react.${reactMode}.js` },
					{ name: 'react-dom', var: 'ReactDOM', path: `umd/react-dom.${reactMode}.js` },
					{ name: 'react-router-dom', var: 'ReactRouterDOM', path: `umd/react-router-dom${min}.js` },
					{ name: 'react-markdown', var: 'ReactMarkdown', path: 'umd/react-markdown.js' },
					// { name: 'antd', path: `dist/antd${min}.js`, style: `dist/antd${min}.css` }
				],
				prod, publicPath: '/node_modules'
			}),
			new MiniCssExtractPlugin({
				filename: '[name].[hash:8].css',
				chunkFilename: '[name].chunk.[chunkhash:8].css'
			})
		]
	}

	if (dev) {
		const API_PORT = require('config').get('port')
		config.devtool = 'inline-source-map'
		config.devServer = {
			historyApiFallback: true,
			proxy: {
				'/api': {
					secure: false,
					changeOrigin: true,
					target: argv.server || `http://localhost:${API_PORT}`
				},
				'/socket.io': {
					ws: true,
					secure: false,
					changeOrigin: true,
					target: argv.server || `ws://localhost:${API_PORT}`
				}
			}
		}
	}

	if (prod) {
		config.plugins.push(
			new OptimizeCssAssetsPlugin()
		)
	}

	return config
}
