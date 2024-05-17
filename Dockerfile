FROM golang:1.22

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go mod tidy

RUN mkdir -p /app/bin
RUN go build -o consumer cmd/consumer/consumer.go

ENV SENDGRID_API_KEY=${SENDGRID_API_KEY}

CMD ["./consumer"]