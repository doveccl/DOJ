FROM node:8
COPY . /home/doj
WORKDIR /home/doj
RUN yarn && yarn build && yarn --production
RUN apt-get update
RUN wget https://github.com/quark-zju/lrun/releases/download/v1.1.4/lrun_1.1.4_amd64.deb
RUN apt-get install -y libseccomp2
RUN dpkg -i lrun_1.1.4_amd64.deb && rm lrun_1.1.4_amd64.deb
