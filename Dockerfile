FROM golang:alpine AS build

WORKDIR /go/src/github.com/urcomputeringpal/kubevalidator
COPY ./vendor ./vendor
RUN go install -v ./vendor/...
COPY . .
RUN go test -v github.com/urcomputeringpal/kubevalidator/...
RUN go install -v github.com/urcomputeringpal/kubevalidator


FROM alpine
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=build /go/bin/kubevalidator /go/bin/kubevalidator
ENTRYPOINT ["/go/bin/kubevalidator"]
