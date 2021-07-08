FROM golang:alpine AS build

ARG race
ARG plugin_type=shared

RUN apk update && \
    apk add yarn alpine-sdk upx

RUN go get -v github.com/gobuffalo/packr/v2/packr2

WORKDIR /squircy

COPY . .

RUN go get -v ./...

RUN make clean dist RACE=${race} PLUGIN_TYPE=${plugin_type}


FROM alpine:latest

RUN apk update && \
    apk add ca-certificates curl gnupg yarn

COPY config.toml.dist /home/squircy/.squircy/config.toml

COPY package.json /home/squircy/.squircy/scripts/package.json

RUN cd /home/squircy/.squircy/scripts && \
    yarn install

RUN adduser -D -h /home/squircy squircy

RUN chown -R squircy: /home/squircy

USER squircy

WORKDIR /squircy

COPY --from=build /squircy/out/squircy_linux_amd64 /bin/squircy

COPY --from=build /squircy/out/*.so /squircy/plugins/

CMD /bin/squircy -interactive
