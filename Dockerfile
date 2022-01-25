FROM node:16
# copy files & change dir
COPY ./dist /doj
WORKDIR /doj
# entry point
ENTRYPOINT [ "/usr/bin/node", "doj.js" ]
