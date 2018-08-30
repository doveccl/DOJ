const path = require('path')
const config = require('config')
const webpack = require('webpack')
const HtmlWebpackPlugin = require('html-webpack-plugin')

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
			filename: '[name].js',
			chunkFilename: 'vendor.[name].js'
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
		plugins: [
			new HtmlWebpackPlugin({
				title: 'DOJ',
				favicon: 'web/logo.png',
				meta: {
					viewport: 'width=device-width, initial-scale=1'
				}
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

	if (prod) {
		config.optimization = {
			splitChunks: {
				chunks: "all",
				cacheGroups: {
					default: false,
					react: {
						name: 'react',
						test: /react/,
						priority: 10
					},
					antd: {
						name: 'antd',
						test: /antd/,
						priority: 5
					},
					other: {
						name: 'other',
						test: /[\\/]node_modules[\\/]/,
						priority: -10
					}
				}
			}
		}
	}

	return config
}
