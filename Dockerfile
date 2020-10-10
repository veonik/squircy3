FROM golang:buster AS build

ARG race

WORKDIR /squircy

COPY . .

RUN go get -v github.com/gobuffalo/packr/v2/... && \
    go install github.com/gobuffalo/packr/v2

RUN go get -v ./...

RUN make clean all RACE=${race}


FROM debian:buster-slim

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y ca-certificates curl gnupg

RUN curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - && \
    echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list && \
    apt-get update && \
    apt-get install -y yarn

COPY config.toml.dist /home/squircy/.squircy/config.toml

COPY package.json /home/squircy/.squircy/scripts/package.json

RUN cd /home/squircy/.squircy/scripts && \
    yarn install

RUN useradd -d /home/squircy squircy

RUN chown -R squircy: /home/squircy

USER squircy

WORKDIR /squircy

COPY --from=build /squircy/out/squircy /bin/squircy

COPY --from=build /squircy/out/*.so /squircy/plugins/

CMD /bin/squircy -interactive
