FROM gliderlabs/alpine:3.2
RUN apk --update add bash curl go git mercurial

RUN curl -Ls https://github.com/progrium/execd/releases/download/v0.1.0/execd_0.1.0_Linux_x86_64.tgz \
    | tar -zxC /bin \
  && curl -Ls https://github.com/progrium/entrykit/releases/download/v0.1.0/entrykit_0.1.0_Linux_x86_64.tgz \
    | tar -zxC /bin \
  && curl -s https://get.docker.io/builds/Linux/x86_64/docker-1.6.1 > /bin/docker \
  && chmod +x /bin/docker \
  && /bin/entrykit

ADD ./data /tmp/data
ADD ./scripts /bin/

ENV GOPATH /go
COPY . /go/src/github.com/progrium/envy
WORKDIR /go/src/github.com/progrium/envy/cmd
RUN go get && go build -o /bin/envyd

VOLUME /data
EXPOSE 22 80
#ENV CODEP_EXECD /bin/execd -e -k /tmp/data/id_host /bin/authenv /bin/enterenv
#ENV CODEP_ENVY /bin/envy
#CMD ["/bin/entrykit", "-e"]
CMD ["/bin/execd", "-e", "-k", "/tmp/data/id_host", "/bin/authenv", "/bin/enterenv"]
