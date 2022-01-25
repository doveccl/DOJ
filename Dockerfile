FROM node:lts-slim
# install deps
RUN apt-get update && apt-get install -y gcc wget
RUN wget https://github.com/quark-zju/lrun/releases/download/v1.1.4/lrun_1.1.4_amd64.deb
RUN dpkg -i lrun_1.1.4_amd64.deb && rm lrun_1.1.4_amd64.deb && apt-get purge -y wget
# copy files & change dir
COPY ./dist /doj
WORKDIR /doj
# entry point
ENTRYPOINT [ "node", "doj.js" ]
