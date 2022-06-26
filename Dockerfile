FROM ubuntu
WORKDIR node
ARG defid_exec
ADD snapshot.tar.gz /.defi/
COPY $defid_exec defid
ENTRYPOINT ["defid", "-daemon"]