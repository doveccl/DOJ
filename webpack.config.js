const config = require('config')
const WebpackCdnPlugin = require('webpack-cdn-plugin')
const HtmlWebpackPlugin = require('html-webpack-plugin')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const CssMinimizerPlugin = require('css-minimizer-webpack-plugin')
const packageJson = require('./package.json')

module.exports = (_env, argv) => {
	const dev = argv.mode === 'development'
	const reactMode = dev ? 'development' : 'production.min'
	const min = dev ? '' : '.min'

	return {
		entry: {
			app: './web'
		},
		output: {
			publicPath: '/',
			filename: '[name].[chunkhash:8].js'
		},
		module: {
			rules: [
				{
					test: /\.tsx?$/,
					loader: 'ts-loader'
				},
				{
					test: /\.(c|le)ss$/,
					use: [
						MiniCssExtractPlugin.loader,
						'css-loader',
						{
							loader: 'less-loader',
							options: {
								lessOptions: {
									javascriptEnabled: true
								}
							}
						}
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
				inject: 'body',
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
					{ name: 'react-markdown', var: 'ReactMarkdown', path: 'react-markdown.min.js' },
					{ name: 'antd', path: `dist/antd${min}.js`, style: `dist/antd${min}.css` },
					{ name: 'github-markdown-css', cssOnly: true, style: `github-markdown${min}.css` }
				],
				prodUrl: '//cdn.jsdelivr.net/npm/:name@:version/:path',
				prod: argv.mode === 'production',
				publicPath: '/node_modules'
			}),
			new MiniCssExtractPlugin({
				filename: '[name].[chunkhash:8].css'
			})
		],
		devtool: dev && 'inline-source-map',
		devServer: {
			historyApiFallback: true,
			proxy: {
				'/api': {
					secure: false,
					changeOrigin: true,
					target: `http://localhost:${config.get('port')}`
				},
				'/socket.io': {
					ws: true,
					secure: false,
					changeOrigin: true,
					target: `http://localhost:${config.get('port')}`
				}
			}
		},
		optimization: {
			minimize: argv.mode === 'production',
			minimizer: [new CssMinimizerPlugin(), '...']
		}
	}
}
