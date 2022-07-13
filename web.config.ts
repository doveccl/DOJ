import webpack from 'webpack'
import WebpackCdnPlugin from 'webpack-cdn-plugin'
import HtmlWebpackPlugin from 'html-webpack-plugin'
import MiniCssExtractPlugin from 'mini-css-extract-plugin'

export default {
	entry: './web',
	output: {
		publicPath: '/',
		path: `${__dirname}/dist/static`,
		filename: 'main.[chunkhash:8].js'
	},
	module: {
		rules: [{
			test: /\.tsx?$/,
			loader: 'ts-loader'
		}, {
			test: /\.(c|le)ss$/,
			use: [MiniCssExtractPlugin.loader, 'css-loader', 'less-loader']
		}]
	},
	resolve: {
		extensions: ['.tsx', '.ts', '.js']
	},
	plugins: [
		new HtmlWebpackPlugin({
			title: 'DOJ',
			inject: 'body',
			favicon: 'web/logo.png'
		}),
		new WebpackCdnPlugin({
			modules: [
				{ name: 'marked', path: `marked.min.js` },
				{ name: 'moment', path: `min/moment.min.js` },
				{ name: 'axios', path: `dist/axios.min.js` },
				{ name: 'katex', style: `dist/katex.min.css` },
				{ name: 'dompurify', var: 'DOMPurify', path: `dist/purify.min.js` },
				{ name: 'ace-builds', var: 'ace', path: 'src-min-noconflict/ace.js' },
				{ name: 'react', var: 'React', path: `umd/react.production.min.js` },
				{ name: 'react-dom', var: 'ReactDOM', path: `umd/react-dom.production.min.js` },
				{ name: 'history', path: `umd/history.production.min.js` },
				{ name: 'react-router', path: `umd/react-router.production.min.js` },
				{ name: 'react-router-dom', var: 'ReactRouterDOM', path: `umd/react-router-dom.production.min.js` },
				{ name: 'antd', path: `dist/antd.min.js`, style: `dist/antd.min.css` },
				{ name: 'github-markdown-css', cssOnly: true, style: 'github-markdown-light.css' },
				{ name: 'highlight.js', var: 'hljs', cdn: '@highlightjs/cdn-assets', path: `highlight.min.js`, style: 'styles/github.min.css' },
				{ name: 'highlightjs-line-numbers.js', var: 'null',  path: 'dist/highlightjs-line-numbers.min.js'},
			]
		}),
		new MiniCssExtractPlugin({
			filename: '[name].[chunkhash:8].css'
		})
	],
	devServer: {
		port: 28080,
		historyApiFallback: true,
		proxy: {
			'/api': {
				secure: false,
				changeOrigin: true,
				target: 'http://localhost:7974'
			},
			'/socket': {
				ws: true,
				secure: false,
				changeOrigin: true,
				target: 'ws://localhost:7974'
			}
		}
	}
} as webpack.Configuration
