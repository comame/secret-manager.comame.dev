FROM ubuntu

RUN useradd -m -s /sbin/nologin app
USER app
WORKDIR /home/app

COPY ./secrets /home/app/secrets

CMD [ "/home/app/secrets" ]
