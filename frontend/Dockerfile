# Step 1: Build the React app
FROM node:18-alpine AS build

WORKDIR /app

# Copy package files and install dependencies
COPY package*.json ./
RUN npm install

# Copy the rest of the app source code
COPY . .

RUN rm -rf build
# Build the app
RUN npm run build

# Step 2: Serve the built app using Nginx
FROM nginx:1.28.0-alpine3.21-slim

# Copy the build output to Nginx's html directory
COPY --from=build /app/build /usr/share/nginx/html

# Use custom Nginx configuration (replaces default one)
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Create a non-root user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Give ownership of Nginx html directory to the non-root user
RUN chown -R appuser:appgroup /usr/share/nginx/html

# Expose HTTP port
EXPOSE 80

# # Switch to the non-root user
# USER appuser

# Run Nginx in the foreground
CMD ["nginx", "-g", "daemon off;"]
