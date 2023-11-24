##
## Build
##
FROM golang:latest AS build

WORKDIR /app

RUN go env -w GOPROXY=https://goproxy.cn,direct

COPY . .

RUN go mod download

RUN GOOS=linux CGO_ENABLED=1 GOARCH=amd64 go build -o /doris-initnetwork

##
## Deploy
##

FROM harbor.kylinserver.top/devops/distroless/base:latest
# FROM scratch

WORKDIR /

COPY manifest /manifest

COPY --from=build /doris-initnetwork /doris-initnetwork

ENTRYPOINT ["/doris-initnetwork"]