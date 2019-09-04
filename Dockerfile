FROM golang:buster AS build

WORKDIR /squircy

COPY . .

RUN go get -v ./...

RUN make clean all


FROM debian:buster-slim

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y ca-certificates curl gnupg

RUN curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - && \
    echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list && \
    apt-get update && \
    apt-get install -y yarn

RUN useradd -d /home/squircy squircy

RUN mkdir -p /home/squircy && \
    chown -R squircy: /home/squircy

USER squircy

WORKDIR /squircy

COPY --from=build /squircy/out/squircy /bin/squircy

COPY --from=build /squircy/out/*.so /squircy/plugins/

CMD /bin/squircy
