import * as Koa from 'koa'

class DataBase {
  public user = {
    get(id: number) {
      return `${id}`
    }
  }
}

declare module 'Koa' {
  interface BaseContext {
    db: DataBase;
  }
}

const app = new Koa()

app.use(async (ctx, next) => {
  ctx.body = '233'
  console.log(ctx.db.user)
})

app.listen(8080)
console.log('server running ...')
