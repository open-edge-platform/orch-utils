# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

# Use distroless as the final base image
FROM gcr.io/distroless/static-debian12:nonroot@sha256:188ddfb9e497f861177352057cb21913d840ecae6c843d39e00d44fa64daa51c
USER nobody

# Now copy the static shell into base image.
COPY --from=busybox:1.37.0-uclibc /bin/sh /bin/sh

COPY --from=bitnami/kubectl:1.31.1 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl

# Set the working directory in the container
WORKDIR /app

ARG IMAGE_NAME
ARG NAME

COPY datamodel_installer.sh /datamodel_installer.sh
COPY build/crds /crds
COPY build/clusterroles /clusterroles

COPY IMAGE /IMAGE
COPY NAME /NAME

CMD ["/bin/sh","/datamodel_installer.sh"]
