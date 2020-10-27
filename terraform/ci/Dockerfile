ARG ATLANTIS_IMAGE=runatlantis/atlantis:v0.15.1
FROM $ATLANTIS_IMAGE

# copy a terraform binary of the version you need
#COPY terragrunt /usr/local/bin/terragrunt

# copy certificates
COPY --chown=atlantis tls /etc/atlantis/tls