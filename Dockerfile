FROM ubuntu:latest
LABEL authors="esha"
COPY . .
ENTRYPOINT ["./client-go"]