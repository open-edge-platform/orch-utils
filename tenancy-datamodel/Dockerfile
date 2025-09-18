# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Use distroless as the final base image
FROM gcr.io/distroless/static-debian12:nonroot
USER nobody

# Now copy the static shell into base image.
COPY --from=busybox:1.37.0-uclibc /bin/sh /bin/sh

COPY --from=rancher/kubectl:v1.31.1 /bin/kubectl /usr/local/bin/kubectl

# Set the working directory in the container
WORKDIR /app

COPY datamodel_installer.sh /datamodel_installer.sh
COPY build/crds /crds
COPY build/clusterroles /clusterroles

COPY IMAGE /IMAGE
COPY NAME /NAME

CMD ["/bin/sh","/datamodel_installer.sh"]
