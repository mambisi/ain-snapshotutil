FROM ubuntu
WORKDIR node
ARG stop_block
ARG volume_name
ENV STOPBLOCK ${stop_block}
RUN useradd --create-home defi && \
    mkdir -p "/${volume_name}" && \
    chown defi:defi "/${volume_name}" && \
    ln -s "/${volume_name}" /home/defi/.defi
RUN adduser defi sudo
ADD snapshot.tar.gz "/${volume_name}"
RUN rm -rf "/${volume_name}/.lock"
RUN rm -rf "/${volume_name}/.walletlock"
RUN rm -rf "/${volume_name}/wallet.dat"
RUN ls -lh "/${volume_name}"
RUN chown -R defi:defi "/${volume_name}"
COPY defid defid
VOLUME ["/${volume_name}"]
RUN chmod +x /node/defid
USER defi:defi
ENTRYPOINT ["/node/defid", "-stop-block=${STOPBLOCK}","-interrupt-block=${STOPBLOCK}"]
EXPOSE 8555 8554 18555 18554 19555 19554