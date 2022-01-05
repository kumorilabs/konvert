FROM gcr.io/distroless/static
COPY konvert /
ENTRYPOINT ["/konvert"]
