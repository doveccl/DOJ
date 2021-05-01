import fs from 'fs-extra'
import axios from 'axios'
import jszip from 'jszip'
import config from 'config'

import { lrunSync } from './run'
import { dataRoot } from './path'

const host: string = config.get('host')
const secret: string = config.get('secret')

function compile(source: string, out: string) {
	return lrunSync({
		maxRealTime: 5,
		passExitcode: true,
		cmd: '/usr/bin/g++',
		args: [source, '-o', out]
	})
}

export async function prepareData(id: string) {
	const dataPath = `${dataRoot}/${id}`
	if (!fs.existsSync(dataPath)) {
		try {
			const zip = await jszip.loadAsync((await axios({
				method: 'GET',
				params: { secret },
				url: `${host}/api/data/${id}`,
				responseType: 'arraybuffer'
			})).data)
			for (const name in zip.files) {
				if (name.endsWith('/')) { continue }
				const content = await zip.files[name].async('nodebuffer')
				await fs.outputFile(`${dataPath}/${name}`, content)
			}
			await fs.chmod(dataPath, 0o777)
			await fs.copy(`testlib/testlib.h`, `${dataPath}/testlib.h`)
			if (!fs.existsSync(`${dataPath}/checker.cpp`)) {
				await fs.copy(`testlib/checker.cpp`, `${dataPath}/checker.cpp`)
			}
			let result = compile(`${dataPath}/checker.cpp`, `${dataPath}/checker`)
			if (result.error) throw result.error
			if (result.status !== 0) {
				const { stdout, stderr } = result
				if (stderr) throw new Error(stderr.toString())
				if (stdout) throw new Error(stdout.toString())
				throw new Error('checker compile error')
			}
			if (fs.existsSync(`${dataPath}/interactor.cpp`)) {
				result = compile(`${dataPath}/interactor.cpp`, `${dataPath}/interactor`)
				if (result.status !== 0) {
					const { stdout, stderr } = result
					if (stderr) throw new Error(stderr.toString())
					if (stdout) throw new Error(stdout.toString())
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
