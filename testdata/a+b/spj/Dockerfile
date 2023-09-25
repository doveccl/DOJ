FROM gcc
WORKDIR /src
COPY judge.cc .
RUN g++ -static -O3 judge.cc -o judge

FROM busybox
COPY --from=0 /src/judge /
CMD [ "/judge" ]
