import config from 'config'
import packageJson from './package.json'
import HtmlWebpackPlugin from 'html-webpack-plugin'
import MiniCssExtractPlugin from 'mini-css-extract-plugin'
import CssMinimizerPlugin from 'css-minimizer-webpack-plugin'
const WebpackCdnPlugin = require('webpack-cdn-plugin')

export default function(_: unknown, argv: Record<string, unknown>) {
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
				// prodUrl: '//cdn.jsdelivr.net/npm/:name@:version/:path',
				modules: [
					{ name: 'marked', path: `marked.min.js` },
					{ name: 'moment', path: `min/moment.min.js` },
					{ name: 'axios', path: `dist/axios${min}.js` },
					{ name: 'katex', style: `dist/katex${min}.css` },
					{ name: 'dompurify', var: 'DOMPurify', path: `dist/purify${min}.js` },
					{ name: 'ace-builds', var: 'ace', path: 'src-min-noconflict/ace.js' },
					{ name: 'socket.io-client', var: 'io', path: 'dist/socket.io.js' },
					{ name: 'react', var: 'React', path: `umd/react.${reactMode}.js` },
					{ name: 'react-dom', var: 'ReactDOM', path: `umd/react-dom.${reactMode}.js` },
					{ name: 'history', path: `umd/history.${reactMode}.js` },
					{ name: 'react-router', path: `umd/react-router.${reactMode}.js` },
					{ name: 'react-router-dom', var: 'ReactRouterDOM', path: `umd/react-router-dom.${reactMode}.js` },
					{ name: 'antd', path: `dist/antd${min}.js`, style: `dist/antd${min}.css` },
					{ name: 'github-markdown-css', cssOnly: true, style: 'github-markdown-light.css' },
					{ name: 'highlight.js', var: 'hljs', cdn: '@highlightjs/cdn-assets', path: `highlight${min}.js`, style: 'styles/github.min.css' },
					{ name: 'highlightjs-line-numbers.js', var: 'null',  path: 'dist/highlightjs-line-numbers.min.js'},
				]
			}),
			new MiniCssExtractPlugin({
				filename: '[name].[chunkhash:8].css'
			})
		],
		devtool: dev && 'inline-source-map',
		watchOptions: {
			ignored: /node_modules/,
		},
		devServer: {
			port: 28080,
			historyApiFallback: true,
			proxy: {
				'/api': {
					secure: false,
					changeOrigin: true,
					target: `http://localhost:${config.get('port')}`
				},
				'/wss': {
					ws: true,
					secure: false,
					changeOrigin: true,
					target: `ws://localhost:${config.get('port')}`
				}
			}
		},
		optimization: {
			minimize: argv.mode === 'production',
			minimizer: [new CssMinimizerPlugin(), '...']
		}
	}
}
