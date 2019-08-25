FROM scratch
COPY konvert /
ENTRYPOINT ["/konvert"]
