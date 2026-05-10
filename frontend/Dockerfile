FROM node:18-alpine AS builder

# Add glibc compatibility for SWC compiler
RUN apk add --no-cache libc6-compat

WORKDIR /app

COPY package*.json ./

RUN --mount=type=cache,target=/root/.npm npm ci --legacy-peer-deps

COPY . .

RUN npm run build

FROM node:18-alpine

# Add glibc compatibility for runtime
RUN apk add --no-cache libc6-compat

WORKDIR /app

COPY --from=builder /app/.next ./.next
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json

EXPOSE 3000

CMD ["npm", "start"]
