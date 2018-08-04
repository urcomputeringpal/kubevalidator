FROM golang:alpine AS build

WORKDIR /go/src/github.com/urcomputeringpal/kubevalidator
COPY ./vendor ./vendor
RUN go install -v ./vendor/...
COPY . .
RUN go install -v github.com/urcomputeringpal/kubevalidator


FROM alpine
COPY --from=build /go/bin/kubevalidator /go/bin/kubevalidator
ENTRYPOINT ["/go/bin/kubevalidator"]
