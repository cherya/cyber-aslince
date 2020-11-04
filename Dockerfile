FROM golang
# All these steps will be cached
RUN mkdir /app
WORKDIR /app
COPY go.mod .
COPY go.sum .

# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download
# COPY the source code as the last step
COPY . .

RUN pwd
RUN ls -la

# Build the binary
RUN make build

CMD ["/app/bin/cyberaslince", "--env=production.env"]