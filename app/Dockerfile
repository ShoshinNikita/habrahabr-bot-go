FROM alpine
# Change timezone
RUN apk add --no-cache tzdata
ENV TZ Europe/Moscow
# Copy the binary
RUN mkdir /app
WORKDIR /app
COPY habrahabr-bot .
ENTRYPOINT [ "./habrahabr-bot" ]