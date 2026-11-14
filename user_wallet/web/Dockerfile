# TigerWallet UserWallet Web (Create React App + TypeScript)
#
# CRA inlines REACT_APP_* at BUILD time, so the backend URL must be supplied as
# a build arg — a runtime `environment:` entry would have no effect. Compose
# passes REACT_APP_API_URL through `build.args`.
FROM node:20-alpine AS builder
WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci --no-audit --no-fund || npm install --no-audit --no-fund

COPY . .

ARG REACT_APP_API_URL
ARG REACT_APP_WS_URL
ARG REACT_APP_TENANT_SLUG
ENV REACT_APP_API_URL=$REACT_APP_API_URL
ENV REACT_APP_WS_URL=$REACT_APP_WS_URL
ENV REACT_APP_TENANT_SLUG=$REACT_APP_TENANT_SLUG

# CRA treats warnings as errors when CI=true.
ENV CI=false
RUN npx react-scripts build

FROM nginx:alpine
COPY --from=builder /app/build /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
