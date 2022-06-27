FROM ubuntu
WORKDIR node
ARG defid_exec
ARG stop_block
ADD snapshot.tar.gz /.defi/
COPY $defid_exec defid
ENTRYPOINT ["defid", "-stop-block=${stop_block}","-interrupt-block=${stop_block}" ]