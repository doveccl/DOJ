# DOJ

Online Judge for OI & ACM/ICPC

## Feature

- AJAX based Single Page Application (Web)
- WebSocket submission result query
- Markdown (with TeX and shortcode supports) editor
- Interactive problems
- Special Judge

## Requirement

- NodeJS 8.x or higher (yarn is recommended)
- MongoDB 3.x or higher
- Docker

## Installation

1. pull image

```bash
docker pull doveccl/doj
```

2. copy [./config](./config) folder and edit your own configurations

- create file `production.json` and add configurations to override `default.json`
- `judger.json` and `server.json` are configuration templates
- for security, please set an unique `secret`

3. simply run command below to start DOJ

```bash
run -it --privileged -p 80:<configured_port,default: 7974> -v /<path_to_your>/config:/doj/config doveccl/doj --server --judger
# or use -d instead of -it for background running
```

4. open http://localhost to test your new OJ, initial root user `admin/admin` could be used for management

## Development

1. clone DOJ to local
2. use `yarn install` or `npm install` to install dependencies
3. use `yarn dev:web` or `npm run dev:web` for web debugging
4. use `yarn dev --server` or `npm run dev --server` for server debugging
5. use `yarn dev --judger` or `npm run dev --judger` for judger debugging
6. you can debug server together with judger by flag `--server --judger`
7. Linux is required for judger debugging
8. to build project, run `yarn build` or `npm run build`

## Separate server and judgers

`server` is the gate of database

single server with multi-judgers is an efficient way to handle with plenty of submissions

- run `docker <...args> doveccl/doj --server` to setup single server without judgers
- run `docker <...args> doveccl/doj --judgers` on several machines to setup multi-judgers without server
- server and judgers should share the same `secret` configuration field
