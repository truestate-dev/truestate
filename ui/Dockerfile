# ── Build stage ───────────────────────────────────────────────────────────────
FROM harbor.apps.vjinx.de/dockerhub/library/node:20-alpine AS build

WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install --prefer-offline

COPY . .
RUN npm run build

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM harbor.apps.vjinx.de/dockerhub/library/nginx:alpine

COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
