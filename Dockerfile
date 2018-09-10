FROM scratch
ADD main /
ADD "brokerConfig/service-config.json" "brokerConfig/service-config.json"
CMD ["/main"]