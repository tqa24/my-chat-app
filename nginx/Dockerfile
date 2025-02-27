FROM nginx:alpine

# Add custom configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Reduce Nginx worker processes for lower memory usage
ENV NGINX_WORKER_PROCESSES=1
ENV NGINX_WORKER_CONNECTIONS=512

# Create cache directories
RUN mkdir -p /var/cache/nginx/client_temp \
    && mkdir -p /var/cache/nginx/proxy_temp \
    && mkdir -p /var/cache/nginx/fastcgi_temp \
    && mkdir -p /var/cache/nginx/uwsgi_temp \
    && mkdir -p /var/cache/nginx/scgi_temp \
    && chown -R nginx:nginx /var/cache/nginx

# Use non-root user
USER nginx

# Expose HTTP port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s CMD wget --quiet --tries=1 --spider http://localhost/ || exit 1
