FROM golang
WORKDIR /src
COPY judge.go .
RUN go build judge.go

FROM busybox
WORKDIR /judge
COPY --from=0 /src/judge .
CMD [ "./judge" ]
