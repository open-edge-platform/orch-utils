# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

# Use distroless as the final base image
FROM gcr.io/distroless/static-debian12:nonroot@sha256:a9329520abc449e3b14d5bc3a6ffae065bdde0f02667fa10880c49b35c109fd1
USER nobody

# Now copy the static shell into base image.
COPY --from=busybox:1.37.0-uclibc /bin/sh /bin/sh

COPY --from=alpine/kubectl:1.35.1 /usr/local/bin/kubectl /usr/local/bin/kubectl

# Set the working directory in the container
WORKDIR /app

COPY datamodel_installer.sh /datamodel_installer.sh
COPY build/crds /crds
COPY build/clusterroles /clusterroles

COPY IMAGE /IMAGE
COPY NAME /NAME

CMD ["/bin/sh","/datamodel_installer.sh"]
