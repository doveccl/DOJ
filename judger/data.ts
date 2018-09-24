import axios from 'axios'
import * as config from 'config'
import * as fs from 'fs-extra'
import * as jszip from 'jszip'
import * as path from 'path'

import { lrunSync } from './run'

const host: string = config.get('host')
const secret: string = config.get('secret')

const cache = path.join(__dirname, '../.cache')
const testlib = path.join(__dirname, 'testlib')

const compile = (source: string, out: string) => {
	return lrunSync({
		cmd: 'g++',
		maxRealTime: 5,
		passExitcode: true,
		args: [ source, '-o', out ]
	})
}

export const prepareData = async (id: string) => {
	const dataPath = `${cache}/${id}`
	if (!await fs.pathExists(dataPath)) {
		try {
			const { data } = await axios({
				method: 'GET',
				params: { secret },
				url: `${host}/api/data/${id}`,
				responseType: 'arraybuffer'
			})
			const zip = await jszip.loadAsync(data)
			for (const name in zip.files) {
				if (name.endsWith('/')) { continue }
				await fs.outputFile(`${dataPath}/${name}`, zip.files[name])
			}
			fs.copyFileSync(`${testlib}/testlib.h`, `${dataPath}/testlib.h`)
			if (!await fs.pathExists(`${dataPath}/checker.cpp`)) {
				fs.copyFileSync(`${testlib}/checker.cpp`, `${dataPath}/checker.cpp`)
			}
			let result = compile(`${dataPath}/checker.cpp`, `${dataPath}/checker`)
			if (result.status !== 0) {
				const { stdout, stderr } = result
				if (stderr) { throw new Error(stderr.toString()) }
				if (stdout) { throw new Error(stdout.toString()) }
				throw new Error('checker compile error')
			}
			if (await fs.pathExists(`${dataPath}/interactor.cpp`)) {
				result = compile(`${dataPath}/interactor.cpp`, `${dataPath}/interactor`)
				if (result.status !== 0) {
					const { stdout, stderr } = result
					if (stderr) { throw new Error(stderr.toString()) }
					if (stdout) { throw new Error(stdout.toString()) }
					throw new Error('interactor compile error')
				}
			}
		} catch (e) {
			await fs.remove(dataPath)
			throw e
		}
	}
	return dataPath
}
