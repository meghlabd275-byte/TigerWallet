# TigerWallet ProjectParty API Dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app/project_party

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /project-party-api ./go/cmd

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/project_party/project-party-api .

EXPOSE 8106

CMD ["./project-party-api"]
