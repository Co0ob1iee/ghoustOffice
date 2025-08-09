#!/usr/bin/env bash
# minimal install stub for patching
cat > /opt/stack/docker-compose.yml <<'EOF'
services:
  core-api:
    build: /opt/stack/core-api
    container_name: core-api
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock
    - ${AUTHELIA_DIR}:/authelia
  environment:
    - PROVISIONER_BASE=https://${MAIN_DOMAIN}
    - AUTHELIA_USERS=/authelia/users_database.yml
    - AUTHELIA_CONTAINER=authelia
    - DOCKER_SOCK=/var/run/docker.sock
    labels:
      - traefik.enable=true
EOF
