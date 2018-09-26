FROM ubuntu:18.04
# copy files & change dir
COPY . /doj
WORKDIR /doj
# install dependencies & build
RUN apt-get update && apt-get install -y wget gnupg && \
	wget -qO- https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - && \
	echo "deb https://dl.yarnpkg.com/debian/ stable main" > /etc/apt/sources.list.d/yarn.list && \
	apt-get update && apt-get install -y gcc python python3 nodejs openjdk-8-jdk-headless libseccomp2 && \
	wget -q https://github.com/quark-zju/lrun/releases/download/v1.1.4/lrun_1.1.4_amd64.deb && \
	dpkg -i lrun_1.1.4_amd64.deb && rm lrun_1.1.4_amd64.deb && \
	yarn install && yarn build && yarn --production && \
	apt-get purge -y wget gnupg yarn && apt-get autoremove -y && \
	rm -f /etc/apt/sources.list.d/yarn.list && rm -rf /var/lib/apt/lists/*
# set env & entry point
ENV NODE_ENV production
ENTRYPOINT [ "/usr/bin/node", "./build" ]
