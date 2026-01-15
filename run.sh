#!/bin/bash

# Quick start script - runs the application with minimal setup

echo "🚀 Starting News Backend..."

# Check if API key is set
if [ -z "$GROQ_API_KEY" ] && [ -z "$OPENAI_API_KEY" ]; then
    echo "⚠️  No API key found. Please set one:"
    echo "export GROQ_API_KEY=your_api_key_here"
    echo ""
    echo "Get a free API key from:"
    echo "- Groq: https://console.groq.com/keys"
    echo "- OpenAI: https://platform.openai.com/api-keys"
    exit 1
fi

# Set default port if not set
if [ -z "$PORT" ]; then
    export PORT=8080
fi

echo "Starting server on port $PORT..."
echo "API will be available at http://localhost:$PORT"
echo ""
echo "Press Ctrl+C to stop"
echo ""

# Run the application
go run main.go
