# Dockerfile

# Global args, set before the first FROM, shared by all stages
ARG NODE_ENV="production"

################################################################################
# Build stage 1 - `yarn build`

FROM node:16-alpine as builder
# Import our shared args
ARG NODE_ENV

# Cache node_modules for as long as possible
COPY cmd/gqld/package.json cmd/gqld/package-lock.json /app/
WORKDIR /app/
RUN npm install --production=false

# Copy over the server source code
COPY cmd/gqld/tsconfig.json /app/
COPY cmd/gqld/src/ /app/src/

# Finally run the build script
RUN npm run build

################################################################################
# Build stage 2 - COPY the relevant things (multiple steps)

FROM node:16-alpine as clean
# Import our shared args
ARG NODE_ENV

# Copy over selectively just the tings we need, try and avoid the rest
COPY --from=builder /app/package.json /app/package-lock.json /app/
COPY --from=builder /app/dist/ /app/dist/

################################################################################
# Build stage FINAL - COPY everything, once, and then do a clean `yarn install`

FROM node:16-alpine
# Import our shared args
ARG NODE_ENV

EXPOSE 3000
WORKDIR /app/
# Copy everything from stage 2, it's already been filtered
COPY --from=clean /app/ /app/

# Install yarn ASAP because it's the slowest
RUN npm install --production=true

LABEL description="My PostGraphile-powered server"

# You might want to disable GRAPHILE_TURBO if you have issues
ENV GRAPHILE_TURBO=1
ENV NODE_ENV=$NODE_ENV
ENTRYPOINT npm start