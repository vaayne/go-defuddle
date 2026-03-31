const path = require('path');
const TerserPlugin = require('terser-webpack-plugin');

module.exports = {
	mode: 'production',
	entry: './internal/js/bundle-entry.js',
	output: {
		path: path.resolve(__dirname, 'internal', 'js'),
		filename: 'defuddle-bundle.js',
		library: {
			type: 'self',
		},
		globalObject: 'globalThis'
	},
	target: 'web',
	resolve: {
		extensions: ['.tsx', '.ts', '.js'],
		alias: {
			'./elements/math': path.resolve(__dirname, 'defuddle', 'src', 'elements', 'math.core.ts'),
		}
	},
	module: {
		rules: [
			{
				test: /\.ts$/,
				use: [
					{
						loader: 'ts-loader',
						options: {
							configFile: path.resolve(__dirname, 'tsconfig.json'),
							transpileOnly: true,
						}
					}
				],
				exclude: /node_modules/
			}
		]
	},
	optimization: {
		minimize: true,
		minimizer: [
			new TerserPlugin({
				terserOptions: {
					output: { ascii_only: true }
				}
			})
		]
	},
	externals: {
		'mathml-to-latex': 'mathml-to-latex',
		'temml': 'temml'
	}
};
