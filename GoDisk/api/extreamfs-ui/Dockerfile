FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build -- --configuration=production

FROM nginx:1.27-alpine
# Copia el bundle de navegador sin depender del nombre del proyecto
COPY --from=builder /app/dist/*/browser /usr/share/nginx/html
# Si tu build no genera /browser, usa esta alternativa:
# COPY --from=builder /app/dist/* /usr/share/nginx/html
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]

ENV NODE_OPTIONS=--max-old-space-size=1024 \
    NG_BUILD_MAX_WORKERS=2 \
    npm_config_audit=false \
    npm_config_fund=false
RUN npm ci --no-audit --no-fund
RUN npm run build -- --configuration=production --no-progress
