FROM gcr.io/distroless/base-debian12:nonroot

COPY ./secrets /usr/local/bin/

EXPOSE 8080
CMD [ "/usr/local/bin/secrets" ]
