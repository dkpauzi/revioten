# Image aplikasi untuk M8.3 (deploy lokal dengan Docker Compose, D-36).
# Tiga tahap: build tampilan web, build binary Go, lalu image jalan yang kecil.
# Versi mengikuti docs/versions.md (R-VER-01): Node 24.22.1, Go 1.27.3.

# 1. Tampilan web (SvelteKit) menjadi berkas statis di web/build/app.
FROM node:24.22.1-bookworm-slim AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
# Skrip prepare butuh berkas proyek, jadi dilewati di sini; build menjalankannya sendiri.
RUN npm ci --ignore-scripts
COPY web/ ./
RUN npm run build

# 2. Binary Go. Hasil build web di-embed ke dalamnya (web/web.go).
FROM golang:1.27.3-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/build ./web/build
RUN CGO_ENABLED=0 go build -trimpath -o /out/revioten ./cmd/revioten

# 3. Image jalan: hanya binary, sebagai pengguna tanpa hak root.
# Image ini tidak berisi Chrome, jadi cetak PDF tidak tersedia di sini;
# cetak Excel tetap jalan (D-172, ASUMSI E-120).
FROM debian:bookworm-slim
RUN useradd --system --uid 10001 --home /app revioten \
    && mkdir -p /data/attachments \
    && chown -R revioten /data
USER revioten
WORKDIR /app
COPY --from=build /out/revioten /app/revioten
ENV ATTACHMENT_DIR=/data/attachments
EXPOSE 8080
# Di dalam container server mendengarkan semua alamat; hanya Caddy yang menjangkaunya.
ENTRYPOINT ["/app/revioten"]
CMD ["-addr", "0.0.0.0:8080"]
