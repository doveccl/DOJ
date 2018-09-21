import axios from 'axios'
import * as config from 'config'
import * as fs from 'fs-extra'
import * as jszip from 'jszip'

const host: string = config.get('host')
const secret: string = config.get('secret')

export const prepareData = async (id: string) => {
	if (!await fs.pathExists(`.cache/${id}`)) {
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
				fs.outputFile(`.cache/${id}/${name}`, zip.files[name])
			}
		} catch (e) {
			await fs.remove(`.cache/${id}`)
		}
	}
}
