FROM node:lts-slim
CMD [ "node", "-e", "process.exit(process.env.code && fs.readFileSync(0, 'utf8').trim() === process.env.code.trim() ? 0 : 1)" ]
