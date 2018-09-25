FROM ubuntu:18.04
# install dependencies
RUN apt-get update && apt-get install -y \
	gcc \
	python \
	python3 \
	fp-compiler \
	nodejs \
	npm \
	wget \
	libseccomp2
# install lrun
RUN wget https://github.com/quark-zju/lrun/releases/download/v1.1.4/lrun_1.1.4_amd64.deb && \
	dpkg -i lrun_1.1.4_amd64.deb && rm lrun_1.1.4_amd64.deb
# build doj
COPY . /doj
WORKDIR /doj
RUN npm install && npm run build && npm prune --production
# setup mirrorfs
RUN mkdir /doj_tmp && lrun-mirrorfs --name doj --setup judger/mirrorfs
