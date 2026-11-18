# openhongguo-signer: the real libmetasec_ml.so under unidbg.
# Build from tools/openhongguo-signer:  docker build -f deploy/Dockerfile .
#
# The jar is pure Java with unicorn natives for linux amd64 and arm64
# inside, so the build stage runs on the build host's platform and only the
# runtime stage follows --platform.
FROM --platform=$BUILDPLATFORM eclipse-temurin:21-jdk AS build
WORKDIR /src
COPY gradlew settings.gradle.kts build.gradle.kts ./
COPY gradle gradle
RUN ./gradlew --no-daemon -q help
COPY src src
RUN ./gradlew --no-daemon -q installDist

FROM eclipse-temurin:21-jre
RUN useradd --system --uid 10001 --home-dir /app openhongguo
COPY --from=build /src/build/install/openhongguo-signer /app
USER openhongguo
WORKDIR /app
ENV SIGNER_HOST=0.0.0.0 \
    SIGNER_PORT=19830 \
    JAVA_OPTS="-XX:MaxRAMPercentage=75 -Djava.io.tmpdir=/tmp"
EXPOSE 19830
# /ready answers once the SDK handle exists. Startup takes 10 to 20 seconds.
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD bash -c 'exec 3<>/dev/tcp/127.0.0.1/${SIGNER_PORT} && printf "GET /ready HTTP/1.0\r\n\r\n" >&3 && grep -q "\"ready\":true" <&3'
ENTRYPOINT ["/app/bin/openhongguo-signer"]
