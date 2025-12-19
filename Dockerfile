FROM scratch
COPY cmd/device-lpmp /device-lpmp
COPY cmd/res /res
EXPOSE 59908

ENTRYPOINT ["/device-lpmp"]
CMD ["-cp=keeper.http://edgex-core-keeper:59890","--registry"]
