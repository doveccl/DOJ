# DOJ

Online Judge for OI & ACM/ICPC

## Installation

0. install `node` and `mongodb`

1. download doj [release](https://github.com/doveccl/DOJ/releases)

## Usage

(optional) create `config.json` with part of [default config](server/util/config.ts)

### standalone-server

```sh
node doj.js --server --judger
```

### multi-server

all server should share same `secret` config

- Server A

```sh
node doj.js --server
```

- Server B/C/D/...

```sh
node doj.js --judger
```

