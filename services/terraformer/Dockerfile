FROM docker.io/library/golang:1.24.5 AS build

ARG TARGETARCH

# download and unzip kube-one binary
RUN apt-get -qq update && apt-get -qq install unzip
RUN VERSION=1.6.2 && \
    wget -q https://github.com/opentofu/opentofu/releases/download/v${VERSION}/tofu_${VERSION}_linux_$TARGETARCH.zip && \
    unzip -qq tofu_${VERSION}_linux_$TARGETARCH.zip -d tofu

#Unset the GOPATH
ENV GOPATH=

#First, copy go.mod and go.sum to prevent uneccesary download of modules
COPY go.mod .
COPY go.sum .

#Check if any modules need downloading
RUN go mod download

#Copy all files apart from the ones in .dockerignore
COPY . .

#Change the directory
WORKDIR /go/services/terraformer/server

#Compile the golang code to /out, CGO_ENABLE=0 removes cross compile dependencies
RUN CGO_ENABLED=0 go build

FROM docker.io/library/alpine:3.20
#Add repository label
LABEL org.opencontainers.image.source "https://github.com/berops/claudie"
#Add image name as a label
LABEL org.opencontainers.image.base.name "docker.io/library/alpine"
#Add description to the image
LABEL org.opencontainers.image.description "Image for Terraformer from Claudie"

#Copy the binaries to empty base image
COPY --from=build /go/tofu/tofu /usr/local/bin

COPY --from=build /go/services/terraformer/server/server /bin/services/terraformer/server/server

COPY --from=build /usr/bin/git /usr/bin/git

RUN apk --no-cache add bash git

#Run server
WORKDIR /bin
ENTRYPOINT [ "./services/terraformer/server/server" ]

#NOTE: We cannot use scratch image for our dockerfile since we are using shell commands to execute commands inside the code
