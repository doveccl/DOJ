export interface IResponse {
  success: boolean
  message?: string
  data?: any
}

export default function(
  response: Promise<any>,
  callback?: (data: any) => any
) {
  return new Promise<any>((resolve, reject) => {
    response.then((res) => {
      const d: IResponse = res.data
      if (d.success) {
        if (callback) {
          callback(d.data)
        }
        resolve(d.data)
      } else {
        reject(d.message)
      }
    })
  })
}
