FROM ubuntu
WORKDIR node
ARG stop_block
ADD snapshot.tar.gz /.defi/
COPY defid defid
RUN chmod -x ./defid
ENTRYPOINT ["./defid", "-stop-block=${stop_block}","-interrupt-block=${stop_block}" ]