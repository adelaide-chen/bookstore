FROM golang:latest as builder
LABEL Description="bookstore"
ARG LOCATION=/
WORKDIR ${LOCATION}

RUN go get -v go.mongodb.org/mongo-driver/mongo
COPY bookstore.go .
RUN go build -a -installsuffix cgo -o app .

FROM alpine:latest
ARG LOCATION=/
EXPOSE 8080
# WORKDIR /root/
COPY --from=builder ${LOCATION} .
CMD ["./app"]