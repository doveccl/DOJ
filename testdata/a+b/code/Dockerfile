FROM gcc
WORKDIR /src
COPY source main.cc
RUN g++ -static -O2 main.cc -o main

FROM busybox
WORKDIR /test
COPY --from=0 /src/main .
CMD [ "./main" ]
