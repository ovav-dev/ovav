FROM node:20-alpine

WORKDIR /app

# Instalar pnpm
RUN npm install -g pnpm

# Copiar archivos
COPY package*.json ./
RUN pnpm install

# Variables de entorno
ENV NEXT_TELEMETRY_DISABLED=1
ENV NEXT_PUBLIC_API_URL=http://host.docker.internal:8080/api

# Exponer puerto
EXPOSE 3000

# Comando
CMD ["pnpm", "dev"]
