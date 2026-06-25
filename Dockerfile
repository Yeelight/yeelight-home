FROM gcr.io/distroless/static-debian12:nonroot

COPY yeelight-home /usr/local/bin/yeelight-home

ENTRYPOINT ["/usr/local/bin/yeelight-home"]
CMD ["version"]
