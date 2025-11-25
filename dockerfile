# =========================
# Etapa de build
# =========================
FROM golang:1.24.0 AS builder
# Si usás otra versión de Go, cambiá 1.22 por la tuya.

WORKDIR /app

# Copiamos sólo los archivos de módulos primero para aprovechar cache
COPY go.mod go.sum ./
RUN go mod download

# Copiamos el resto del código
COPY . .

# Compilamos binario estático
RUN CGO_ENABLED=0 GOOS=linux go build -o mentorly-backend .

# =========================
# Etapa de runtime
# =========================
FROM alpine:3.19

# Para que los errores / logs salgan bien con timezones
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copiamos el binario desde la etapa de build
COPY --from=builder /app/mentorly-backend .

# Puerto donde escucha tu servidor (main.go usa :8080)
ENV PORT=8080
EXPOSE 8080

# IMPORTANTE: las vars sensibles (DATABASE_URL, JWT_SECRET, MP_ACCESS_TOKEN, FRONTEND_URL, etc.)
# NO las copies al contenedor; pasalas como env vars en el deploy.
CMD ["./mentorly-backend"]
