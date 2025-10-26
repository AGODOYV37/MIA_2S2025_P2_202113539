# ----------- Builder: Node -----------
FROM node:20-alpine AS builder
WORKDIR /app

# Si quieres desactivar audit/fund, hazlo aquí (en builder)
ENV npm_config_audit=false \
    npm_config_fund=false

COPY package*.json ./
RUN npm ci

COPY . .
# Build del frontend (producción)
RUN npm run build -- --configuration=production --no-progress

# ----------- Runtime: Nginx -----------
FROM nginx:1.27-alpine
# Copia artefactos de navegador
COPY --from=builder /app/dist/*/browser /usr/share/nginx/html
# Tu config de Nginx con el proxy /api
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
