FROM --platform=linux/arm64 alpine:3.20
RUN apk add --no-cache dumb-init ca-certificates tzdata \
    && update-ca-certificates
    
WORKDIR /app
COPY cmd/device-lpmp /app/device-lpmp
COPY cmd/res /app/res
COPY cmd/configuration.yaml /app/configuration.yaml

RUN chmod +x /app/device-lpmp
EXPOSE 59908

ENTRYPOINT ["dumb-init","--","/app/device-lpmp"]
CMD ["-cp=keeper.http://edgex-core-keeper:59890","--registry"]
