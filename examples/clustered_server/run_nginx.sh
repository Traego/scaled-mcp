#!/bin/bash
# Script to run Nginx with our custom configuration

# Kill any existing Nginx processes
pkill nginx 2>/dev/null

# Run Nginx with our configuration
nginx -c "$(pwd)/nginx.conf" -p "$(pwd)"

echo "Nginx started on port 8000, proxying to backends on 8080 and 8081"
echo "Press Ctrl+C to stop"

# Wait for Nginx to exit
wait
