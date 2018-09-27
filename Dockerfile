FROM ubuntu:18.04
# copy files & change dir
COPY . /doj
WORKDIR /doj
# change sources
# RUN sed -i 's/archive.ubuntu.com/mirrors.ustc.edu.cn/g' /etc/apt/sources.list
# install dependencies & build
RUN apt-get update && apt-get install -y wget gnupg && \
	wget -qO- https://dl.yarnpkg.com/debian/pubkey.gpg | gpg --dearmor > /etc/apt/trusted.gpg.d/yarn.gpg && \
	echo "deb https://dl.yarnpkg.com/debian/ stable main" > /etc/apt/sources.list.d/yarn.list && \
	apt-get update && apt-get install -y gcc g++ python python3 nodejs openjdk-8-jdk-headless libseccomp2 yarn && \
	wget -q https://github.com/quark-zju/lrun/releases/download/v1.1.4/lrun_1.1.4_amd64.deb && \
	dpkg -i lrun_1.1.4_amd64.deb && rm lrun_1.1.4_amd64.deb && \
	yarn install && yarn build && yarn --production && \
	apt-get purge -y wget gnupg yarn && apt-get autoremove -y && \
	rm -rf /etc/apt/sources.list.d/yarn.list /etc/apt/trusted.gpg.d/yarn.gpg /var/lib/apt/lists/*
# set env & entry point
ENV NODE_ENV production
ENTRYPOINT [ "/usr/bin/node", "./build" ]
