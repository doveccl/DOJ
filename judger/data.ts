import fs from 'fs'
import axios from 'axios'
import jszip from 'jszip'

import { lrunSync } from './run'
import { dataRoot } from './path'
import config from '../config'

function compile(source: string, out: string) {
  return lrunSync({
    maxRealTime: 5,
    passExitcode: true,
    cmd: '/usr/bin/g++',
    args: [source, '-o', out]
  })
}

export async function prepareData(id: string) {
  const { host, secret } = config
  const dataPath = `${dataRoot}/${id}`

  if (!fs.existsSync(dataPath)) {
    try {
      await fs.promises.mkdir(dataPath, { recursive: true })
      const zip = await jszip.loadAsync((await axios({
        method: 'GET',
        params: { secret },
        url: `http://${host}/api/data/${id}`,
        responseType: 'arraybuffer'
      })).data)
      for (const name in zip.files) {
        if (name.endsWith('/')) { continue }
        const content = await zip.files[name].async('nodebuffer')
        await fs.promises.writeFile(`${dataPath}/${name}`, content)
      }
      await fs.promises.chmod(dataPath, 0o777)
      await fs.promises.copyFile(`testlib/testlib.h`, `${dataPath}/testlib.h`)
      if (!fs.existsSync(`${dataPath}/checker.cpp`)) {
        await fs.promises.copyFile(`testlib/checker.cpp`, `${dataPath}/checker.cpp`)
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
      await fs.promises.rm(dataPath, {
        force: true,
        recursive: true
      })
      throw e
    }
  }
  return dataPath
}
