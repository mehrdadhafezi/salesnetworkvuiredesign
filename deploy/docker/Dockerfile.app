# R3 application image. Built FROM a verified release artifact directory
# (scripts/release/build.sh + verify.sh), never from source: the binaries in
# the image are exactly the checksummed artifact. No secret is passed as a
# build argument or baked into a layer; credentials are mounted at runtime
# as files (/run/secrets, *_FILE variables).
#
#   docker build -f deploy/docker/Dockerfile.app -t crm-app:<commit12> <artifact-dir>
FROM gcr.io/distroless/static-debian12@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
ARG CRM_COMMIT=unknown
LABEL org.opencontainers.image.title="crm-app" \
      org.opencontainers.image.revision="${CRM_COMMIT}" \
      org.opencontainers.image.source="https://github.com/mehrdadhafezi/salesnetworkvuiredesign"
COPY --chown=0:0 --chmod=0555 bin/ /opt/crm/bin/
COPY --chown=0:0 --chmod=0444 SHA256SUMS manifest.json /opt/crm/
USER 65532:65532
ENTRYPOINT ["/opt/crm/bin/crm-api"]
