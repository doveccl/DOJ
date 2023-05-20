FROM node:lts-slim AS builder
WORKDIR /src
COPY package.json .
COPY yarn.lock .
RUN yarn
COPY . .
RUN yarn build
RUN yarn --production

FROM node:lts-slim
WORKDIR /doj
RUN apt-get update && \
  apt-get install -y g++ python2 python3 openjdk-17-jdk golang rustc libseccomp2 && \
  rm -rf /var/lib/apt/lists/*
ADD https://github.com/quark-zju/lrun/releases/download/v1.1.4/lrun_1.1.4_amd64.deb lrun.deb
RUN dpkg -i lrun.deb && rm lrun.deb
COPY --from=builder /src/out .
COPY --from=builder /src/dist dist
COPY --from=builder /src/node_modules node_modules
COPY mirrorfs.cfg mirrorfs.cfg
COPY testlib testlib
ENTRYPOINT [ "node", "." ]
