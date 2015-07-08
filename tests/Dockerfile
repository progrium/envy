FROM progrium/dind
RUN apk --update add expect bash openssh curl docker make \
  && ssh-keygen -t rsa -N "" -f /root/.ssh/id_rsa \
  && curl -Ls https://github.com/progrium/basht/releases/download/v0.1.0/basht_0.1.0_Linux_x86_64.tgz \
    | tar -zxC /bin
ENV DOCKER_OPTS -H tcp://0.0.0.0:2376 -H unix:///var/run/docker.sock
ENV DOCKER_HOST tcp://localhost:2376
WORKDIR /envy
ADD . /envy
