FROM golang:1.17 as build-env

WORKDIR /go/src/app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go vet -v ./...
RUN go test ./... -v

RUN CGO_ENABLED=0 go build -o /go/bin/konvert

FROM gcr.io/distroless/static

COPY --from=build-env /go/bin/konvert /
ENTRYPOINT ["/konvert"]
