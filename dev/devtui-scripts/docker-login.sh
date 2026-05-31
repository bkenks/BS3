#!/usr/bin/env bash
# Log in to the GitHub Container Registry (GHCR) before releasing. Interactive —
# prompts for username and password. Use your GitHub username and a Personal
# Access Token (classic) with the `write:packages` scope as the password.
exec docker login ghcr.io
