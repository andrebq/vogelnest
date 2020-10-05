FROM golang:1.14-alpine as glua
RUN apk add git
RUN go get -u github.com/andrebq/glua

FROM golang:1.14-alpine as app
WORKDIR /app/vogelnest
COPY go.mod go.sum /app/vogelnest/
RUN go mod download
COPY . /app/vogelnest/
RUN go build -o vogelnest .

FROM node:lts-alpine as ui
COPY internal/ui /vogelnest-ui
ENV API_ROOT=vogelnest.ep.cluster.amoraes.info
WORKDIR /vogelnest-ui
RUN rm -rf public/build && yarn install && yarn run build

FROM alpine
WORKDIR /opt/vogelnest
COPY --from=glua /go/bin/glua /usr/local/bin
COPY --from=ui /vogelnest-ui/public /opt/vogelnest/static/
COPY --from=app /app/vogelnest/vogelnest .
CMD [ "/opt/vogelnest/vogelnest", "-serve-static", "/opt/vogelnest/static/", "-storage", "/var/data/vogelnest/tweets"]
