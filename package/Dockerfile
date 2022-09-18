FROM registry.suse.com/bci/bci-micro:15.3
RUN echo 'uipluginoperator:x:1000:1000::/home/uipluginoperator:/bin/bash' >> /etc/passwd && \
    echo 'uipluginoperator:x:1000:' >> /etc/group && \
    mkdir /home/uipluginoperator && \
    chown -R uipluginoperator:uipluginoperator /home/uipluginoperator
COPY bin/ui-plugin-operator /usr/bin/
USER uipluginoperator
CMD ["ui-plugin-operator"]
