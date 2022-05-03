FROM golang:1.18.1-alpine3.15 AS build 
ENV CGO_ENABLED 0

RUN apk add git make openssl

WORKDIR /go/src/github.com/implodingduck/aks-http-proxy-append
COPY . .
RUN make app

FROM scratch
WORKDIR /app
COPY --from=build /go/src/github.com/implodingduck/aks-http-proxy-append .
COPY --from=build /go/src/github.com/implodingduck/aks-http-proxy-append/ssl /ssl

CMD ["/app/akshttpproxyappend"]