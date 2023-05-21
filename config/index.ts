import { parse, stringify } from 'yaml'
import { readFileSync, writeFileSync } from 'fs'

const config = parse(`
port: 7974
name: default_judger
host: 127.0.0.1:7974
concurrent: 2
database: mongodb://127.0.0.1:27017/doj
log: debug
secret: doj_secret
registration: true
mail:
  host: smtp.mailtrap.io
  port: 2525
  auth:
    user: 2e35494e193977
    pass: 1a2a2128eb634e
languages:
  - name: C
    suffix: c
    source: main.c
    compile:
      time: 10
      cmd: /usr/bin/gcc
      args: [main.c, -o, main, -O2, -lm, -std=c99, -DONLINE_JUDGE]
    run:
      ratio: 1
      cmd: ./main
  - name: C++
    suffix: cpp
    source: main.cpp
    compile:
      time: 10
      cmd: /usr/bin/g++
      args: [main.cpp, -o, main, -O2, -lm, -std=c++17, -DONLINE_JUDGE]
    run:
      ratio: 1
      cmd: ./main
  - name: Python2
    suffix: py
    source: main.py
    run:
      ratio: 2
      cmd: /usr/bin/python2
      args: [main.py]
  - name: Python3
    suffix: py
    source: main.py
    run:
      ratio: 2
      cmd: /usr/bin/python3
      args: [main.py]
  - name: Java
    suffix: java
    source: Main.java
    compile:
      time: 10
      cmd: /usr/bin/javac
      args: [Main.java]
    run:
      ratio: 2
      cmd: /usr/bin/java
      args: [-Xms128m, -Xmx1g, Main]
  - name: JavaScript
    suffix: js
    source: main.js
    run:
      ratio: 2
      cmd: /usr/local/bin/node
      args: [main.js]
  - name: Go
    suffix: go
    source: main.go
    compile:
      time: 10
      cmd: /usr/bin/go
      args: [build, -o, main, main.go]
    run:
      ratio: 1
      cmd: ./main
  - name: Rust
    suffix: rs
    source: main.rs
    compile:
      time: 10
      cmd: /usr/bin/rustc
      args: [main.rs]
    run:
      ratio: 1
      cmd: ./main
`)

try {
  const conf = readFileSync('config.yaml', 'utf8')
  Object.assign(config, parse(conf))
} catch {
  writeFileSync('config.yaml', stringify(config))
}

export default config
