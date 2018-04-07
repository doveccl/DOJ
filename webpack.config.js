const path = require('path')
const webpack = require('webpack')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const CleanWebpackPlugin = require('clean-webpack-plugin')
const HtmlWebpackPlugin = require('html-webpack-plugin')
const WebpackCdnPlugin = require('webpack-cdn-plugin')

const SERVER_PORT = require('./config').server.port

let DEV_MODE = false
let NPM_CMD = process.env.npm_lifecycle_script
if (/--mode\s+development/.test(NPM_CMD)) {
	DEV_MODE = true
}

let config = {
	entry: './src/index.js',
	output: {
		path: path.join(__dirname, 'dist'),
		filename: 'main.[hash].js'
	},
	module: {
		rules: [
			{
				test: /\.vue$/,
				loader: 'vue-loader'
			},
			{
				test: /\.css$/,
				use: [
					MiniCssExtractPlugin.loader,
					'css-loader'
				]
			},
			{
				test: /\.styl$/,
				use: [
					MiniCssExtractPlugin.loader,
					'css-loader',
					'stylus-loader'
				]
			},
			{
				test: /\.(gif|jpg|png|svg)$/,
				loader: 'url-loader?limit=1024'
			},
			{
				test: /\.(woff|woff2|eot|ttf)$/,
				loader: 'url-loader?limit=1024'
			}
		]
	},
	plugins: [
		new CleanWebpackPlugin(['dist']),
		new MiniCssExtractPlugin({
			filename: '[name].[hash].css',
			chunkFilename: '[id].[chunkhash].css'
		}),
		new HtmlWebpackPlugin({
			template: 'src/index.html',
			minify: {
				removeComments: true,
				collapseWhitespace: true,
				removeAttributeQuotes: true
			}
		})
	]
}

if (DEV_MODE) {
	config.plugins.push(
		new webpack.NamedModulesPlugin(),
		new webpack.HotModuleReplacementPlugin()
	)
	config.resolve = {
		alias: {
			vue: 'vue/dist/vue.js'
		}
	}
	config.devServer = {
		hot: true,
		overlay: {
			warnings: true,
			errors: true
		},
		proxy: {
			'/api': {
				target: `http://localhost:${SERVER_PORT}`,
				changeOrigin: true
			}
		},
		historyApiFallback: true
	}
} else {
	config.output.publicPath = './'
	config.externals = {
		'vue': 'Vue',
		'vue-router': 'VueRouter',
		'iview': 'iview',
		'iview/dist/styles/iview.css': 'null'
	}
	config.plugins.push(
		new WebpackCdnPlugin({
			modules: [
				{
					name: 'vue',
					var: 'Vue',
					path: 'dist/vue.min.js'
				},
				{
					name: 'vue-router',
					var: 'VueRouter',
					path: 'dist/vue-router.min.js'
				},
				{
					name: 'iview',
					path: 'dist/iview.min.js',
					style: 'dist/styles/iview.css'
				}
			]
		})
	)
}

module.exports = config
