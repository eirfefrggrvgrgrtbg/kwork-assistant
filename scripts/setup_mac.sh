#!/bin/bash

# Configuration
EXPECTED_MODEL="gemma4:e4b"

echo "Checking environment for Kwork Assistant..."

# 1. Check Ollama
if ! command -v ollama &> /dev/null; then
    echo "❌ Ollama is not installed."
    echo "To install, visit: https://ollama.com/download or run:"
    echo "brew install ollama"
    exit 1
fi

echo "✅ Ollama is installed."
ollama --version

# 2. Check if Ollama is running
if ! curl -s http://localhost:11434/api/tags > /dev/null; then
    echo "❌ Ollama is not running on http://localhost:11434."
    echo "Please start the Ollama application."
    exit 1
fi
echo "✅ Ollama service is accessible."

# 3. Check for the model
HAS_MODEL=$(ollama list | grep -q "$EXPECTED_MODEL" && echo "yes" || echo "no")

if [ "$HAS_MODEL" = "no" ]; then
    echo "❌ Expected model '$EXPECTED_MODEL' is NOT installed."
    echo "To install it, run the following command:"
    echo "ollama run $EXPECTED_MODEL"
    echo ""
    echo "Note: This will download the model. Run the command manually so you can track the download progress."
    exit 1
fi

echo "✅ Model '$EXPECTED_MODEL' is installed."
echo "Setup checks passed!"
exit 0
