FROM registry.suse.com/bci/bci-micro:15.6
ARG TARGETOS
ARG TARGETARCH
RUN echo 'uipluginoperator:x:1000:1000::/home/uipluginoperator:/bin/bash' >> /etc/passwd && \
    echo 'uipluginoperator:x:1000:' >> /etc/group && \
    mkdir /home/uipluginoperator && \
    chown -R uipluginoperator:uipluginoperator /home/uipluginoperator
COPY dist/ui-plugin-operator_${TARGETOS}_${TARGETARCH} /usr/bin/ui-plugin-operator
USER uipluginoperator
CMD ["ui-plugin-operator"]
